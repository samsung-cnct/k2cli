// Copyright © 2016 Samsung CNCT
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/namesgenerator"
	"golang.org/x/net/context"
)


// Close can throw an err, so to defer to it is risky,
// review http://www.blevesearch.com/news/Deferred-Cleanup,-Checking-Errors,-and-Potential-Problems/
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}

func base64EncodeAuth(auth types.AuthConfig) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(auth); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

func streamLogs(cli *client.Client, resp types.ContainerCreateResponse, ctx context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerLogOpts := types.ContainerLogsOptions{ ShowStdout: true, Follow: true}
	reader, err := cli.ContainerLogs(ctx, resp.ID, containerLogOpts)
	if err != nil {
		log.Fatal(err)
	}

	defer Close(reader)

	if _, err = io.Copy(os.Stdout, reader); err != nil && err != io.EOF {
		log.Fatal(err)
	}
}

func printContainerLogs(cli *client.Client, resp types.ContainerCreateResponse, ctx context.Context) ([]byte, error) {
	containerLogOpts := types.ContainerLogsOptions{ ShowStdout: true, ShowStderr: true}
	out, err := cli.ContainerLogs(ctx, resp.ID, containerLogOpts)
	if err != nil {
		return nil, err
	}

	defer Close(out)

	return ioutil.ReadAll(out)
}

// post cluster help types
type helpType int

const (
	Created   helpType = iota
	Destroyed
	Updated
)

func clusterHelpError(help helpType, clusterConfigFile string) {
	fmt.Println("Some of the cluster state MAY be available:")
	clusterHelp(help, clusterConfigFile)
}

func clusterHelp(help helpType, clusterConfigFile string) {
	if _, err := os.Stat(path.Join(outputLocation, getFirstClusterName(), "admin.kubeconfig")); err == nil {
		fmt.Println("To use kubectl: ")
		fmt.Println(" kubectl --kubeconfig=" + path.Join(
			outputLocation,
			getFirstClusterName(), "admin.kubeconfig") + " [kubectl commands]")
		fmt.Println(" or use 'kraken tool kubectl --config " + clusterConfigFile + " [kubectl commands]'")

		if _, err := os.Stat(path.Join(outputLocation,
			getFirstClusterName(), "admin.kubeconfig")); err == nil {
			fmt.Println("To use helm: ")
			fmt.Println(" export KUBECONFIG=" + path.Join(
				outputLocation,
				getFirstClusterName(), "admin.kubeconfig"))
			fmt.Println(" helm [helm command] --home " + path.Join(
				outputLocation,
				getFirstClusterName(), ".helm"))
			fmt.Println(" or use 'kraken tool helm --config " + clusterConfigFile + " [helm commands]'")
		}
	}

	if _, err := os.Stat(path.Join(outputLocation, getFirstClusterName(), "ssh_config")); err == nil {
		fmt.Println("To use ssh: ")
		fmt.Println(" ssh <node pool name>-<number> -F " + path.Join(outputLocation, getFirstClusterName(), "ssh_config"))
		// This is usage has not been implemented. See issue #49
		//fmt.Println(" or use 'kraken tool --config ssh ssh " + clusterConfigFile + " [ssh commands]'")
	}
}

// Convert dashes to underscore (if any) in cluster name and append to helm_override_ to be able to pull correct env for helm override
func setHelmOverrideEnv(name string) string {
	return fmt.Sprintf("helm_override_%s", strings.Replace(name, "-", "_", -1))
}

func containerEnvironment() []string {
	containerName := getFirstClusterName()

	envs := []string{"ANSIBLE_NOCOLOR=True",
		"DISPLAY_SKIPPED_HOSTS=0",
		"KUBECONFIG=" + path.Join(outputLocation, containerName, "admin.kubeconfig"),
		"HELM_HOME=" + path.Join(outputLocation, containerName, ".helm")}

	envs = appendIfValueNotEmpty(envs, "AWS_ACCESS_KEY_ID")
	envs = appendIfValueNotEmpty(envs, "AWS_SECRET_ACCESS_KEY")
	envs = appendIfValueNotEmpty(envs, "AWS_DEFAULT_REGION")
	envs = appendIfValueNotEmpty(envs, "CLOUDSDK_COMPUTE_ZONE")
	envs = appendIfValueNotEmpty(envs, "CLOUDSDK_COMPUTE_REGION")
	envs = appendIfValueNotEmpty(envs, setHelmOverrideEnv(containerName))

	return envs
}

// append to slice if environment variable (key) has a non-empty value.
func appendIfValueNotEmpty(envs []string, envKey string) []string {
	if env := os.Getenv(envKey); len(env) > 0 {
		return append(envs, envKey+"="+env)
	}

	return envs
}

func makeMounts(clusterConfigPath string) (*container.HostConfig, []string) {
	config_envs := []string{}

	// cluster configuration is always mounted
	var hostConfig *container.HostConfig
	if len(strings.TrimSpace(clusterConfigPath)) > 0 {
		hostConfig = &container.HostConfig{
			Binds: []string{
				clusterConfigPath + ":" + clusterConfigPath,
				outputLocation + ":" + outputLocation},
		}

		deployment := reflect.ValueOf(clusterConfig.Sub("deployment"))
		parseMounts(deployment, hostConfig, &config_envs)

	} else {
		hostConfig = &container.HostConfig{
			Binds: []string{
				outputLocation + ":" + outputLocation},
		}
	}

	return hostConfig, config_envs
}

func parseMounts(deployment reflect.Value, hostConfig *container.HostConfig, config_envs *[]string) {
	switch deployment.Kind() {
	case reflect.Ptr:
		deploymentValue := deployment.Elem()

		// Check if the pointer is nil
		if !deploymentValue.IsValid() {
			return
		}

		parseMounts(deploymentValue, hostConfig, config_envs)

	case reflect.Interface:
		deploymentValue := deployment.Elem()
		parseMounts(deploymentValue, hostConfig, config_envs)

	case reflect.Struct:
		for i := 0; i < deployment.NumField(); i += 1 {
			parseMounts(deployment.Field(i), hostConfig, config_envs)
		}

	case reflect.Slice:
		for i := 0; i < deployment.Len(); i += 1 {
			parseMounts(deployment.Index(i), hostConfig, config_envs)
		}

	case reflect.Map:
		for _, key := range deployment.MapKeys() {
			originalValue := deployment.MapIndex(key)
			parseMounts(originalValue, hostConfig, config_envs)
		}
	case reflect.String:
		reflectedString := fmt.Sprintf("%s", deployment)

		// if the string was an environment variable we need to add it to the config_envs
		regex := regexp.MustCompile(`\$[A-Za-z0-9_]+`)
		matches := regex.FindAllString(reflectedString, -1)
		for _, value := range matches {
			*config_envs = append(*config_envs, strings.Replace(value, "$", "", -1)+"="+os.ExpandEnv(value))
		}

		if _, err := os.Stat(os.ExpandEnv(reflectedString)); err == nil {
			if filepath.IsAbs(os.ExpandEnv(reflectedString)) {
				for _, bind := range hostConfig.Binds {
					if bind == os.ExpandEnv(reflectedString)+":"+os.ExpandEnv(reflectedString) {
						return
					}
				}
				hostConfig.Binds = append(hostConfig.Binds, os.ExpandEnv(reflectedString)+":"+os.ExpandEnv(reflectedString))
			}
		}
	default:
	}
}

func getClient() (*client.Client, error) {
	var httpClient *http.Client

	if dockerClient.isInheritedFromEnvironment() {
		// Rely on Docker's own standard environment handling.
		return client.NewEnvClient()
	} else {
		// Set up an optionally TLS-enabled client, based on non-environment flags.
		// This replicates logic of Docker's `NewEnvClient`, but allows for our
		// command-line argument overrides.
		if dockerClient.isTLSActivated() {

			tlsClientConfig, err := dockerClient.createTLSConfig()
			if err != nil {
				return nil, err
			}

			httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsClientConfig,
				},
			}

		}

		headers := map[string]string{"User-Agent": fmt.Sprintf("engine-api-cli-%s", dockerClient.DockerAPIVersion)}

		return client.NewClient(dockerClient.DockerHost, dockerClient.DockerAPIVersion, httpClient, headers)
	}
}

func getAuthConfig64(cli *client.Client, ctx context.Context) (string, error) {
	authConfig := types.AuthConfig{}
	if len(userName) > 0 && len(password) > 0 {
		imageParts := strings.Split(containerImage, "/")

		if strings.Count(imageParts[0], ".") > 0 {
			authConfig.ServerAddress = imageParts[0]
		} else {
			authConfig.ServerAddress = "index.docker.io"
		}

		authConfig.Username = userName
		authConfig.Password = password

		_, err := cli.RegistryLogin(ctx, authConfig)
		if err != nil {
			return "", nil
		}
	}

	return base64EncodeAuth(authConfig)
}

func pullImage(cli *client.Client, ctx context.Context, base64Auth string) error {

	pullOpts := types.ImagePullOptions{
		RegistryAuth:  base64Auth,
		All:           false,
		PrivilegeFunc: nil,
	}

	pullResponseBody, err := cli.ImagePull(ctx, containerImage, pullOpts)
	if err != nil {
		return err
	}

	defer Close(pullResponseBody)

	// wait until the image download is finished
	dec := json.NewDecoder(pullResponseBody)
	m := map[string]interface{}{}
	for {
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	// if the final stream object contained an error
	if errMsg, ok := m["error"]; ok {
		return fmt.Errorf("%v", errMsg)
	}
	return nil
}

func containerAction(cli *client.Client, ctx context.Context, command []string, krakenlibconfig string) (types.ContainerCreateResponse, int, func(), error) {
	var containerResponse types.ContainerCreateResponse

	hostConfig, config_envs := makeMounts(krakenlibconfig)
	containerConfig := &container.Config{
		Image:        containerImage,
		Env:          append(containerEnvironment(), config_envs...),
		Cmd:          command,
		AttachStdout: true,
		Tty:          true,
	}

	// ^[\\w]+[\\w-. ]*[\\w]+$ is the name requirement for docker containers as of 1.13.0
	//  clusterName can be empty as a valid thing when a user is generating a config so the
	//  hardcoded base portion of the name must satisfy the above regex.
	clusterName := getFirstClusterName()
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, "krakenlib"+clusterName)
	if err != nil {
		return containerResponse, -1, nil, err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return containerResponse, -1, nil, err
	}

	if verbosity {
		backgroundCtx := getContext()
		streamLogs(cli, resp, backgroundCtx)
	}

	statusCode, err := cli.ContainerWait(ctx, resp.ID)
	if err != nil {
		select {
		case <-ctx.Done():
			fmt.Println("Action timed out!")
			return resp, 1, containerRenameOrRemove(cli, resp, clusterName, true, true), nil
		default:
			return containerResponse, -1, nil, err
		}
	}

	return resp, statusCode, containerRenameOrRemove(cli, resp, clusterName, false, false), nil
}

func containerRenameOrRemove(cli *client.Client, resp types.ContainerCreateResponse, clusterName string, doKill bool, forceRemove bool) func() {
	return func() {
		var err error

		if keepAlive {
			if doKill {
				if err = cli.ContainerKill(getContext(), resp.ID, "KILL"); err != nil {
					log.Fatalf("Error clean doing container renaming or removing: %s", err)
				}
			}

			oldContainerName := fmt.Sprintf("k2-%s", clusterName)
			newContainerName := fmt.Sprintf("k2-%s", namesgenerator.GetRandomName(1))

			err = cli.ContainerRename(getContext(), resp.ID, newContainerName)
			if err == nil {
				fmt.Printf("Renamed %s to %s \n", oldContainerName, newContainerName)
			}
		} else {
			removeOpts := types.ContainerRemoveOptions{	RemoveVolumes: false, RemoveLinks: false, Force: forceRemove}
			err = cli.ContainerRemove(getContext(),	resp.ID, removeOpts)
		}

		if err != nil {
			log.Fatalf("Error clean doing container renaming or removing: %s", err)
		}
	}
}

func getContext() (ctx context.Context) {
	return context.Background()
}

func getTimedContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(actionTimeout)*time.Second)
}

func writeLog(logFilePath string, out []byte) error {
	var fileHandle *os.File

	_, err := os.Stat(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {

			// make sure path exists
			err = os.MkdirAll(filepath.Dir(logFilePath), 0777)
			if err != nil {
				return err
			}

			// check if a valid file path
			var d []byte

			if err := ioutil.WriteFile(logFilePath, d, 0644); err != nil {
				return err
			} else {
				os.Remove(logFilePath)
			}

			fileHandle, err = os.Create(logFilePath)
			if err != nil {
				return err
			}
		} else {
			fileHandle, err = os.OpenFile("test.txt", os.O_RDWR, 0666)
			if err != nil {
				return err
			}
		}
	}

	defer Close(fileHandle)

	_, err = fileHandle.Write(out)
	return err
}

func getFirstClusterName() string {
	// only supports first cluster name right now
	if clusters := clusterConfig.Get("deployment.clusters"); clusters != nil {
		firstCluster := clusters.([]interface{})[0].(map[interface{}]interface{})
		if firstCluster["name"] == nil {
			return "cluster-name-missing"
		}
		// should not use type assertion .(string) without verifying interface isnt nil
		return os.ExpandEnv(firstCluster["name"].(string))
	} else {
		return "cluster-name-missing"
	}
}
