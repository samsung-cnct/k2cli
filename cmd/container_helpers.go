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
	"github.com/docker/docker/pkg/tlsconfig"
	"golang.org/x/net/context"
)

var DockerAPIVersion = client.DefaultVersion

// DockerClientConfig provides a simple encapsulation of parameters to construct the Docker API client
type DockerClientConfig struct {
	DockerHost       string
	DockerAPIVersion string
	TLSEnabled       bool
	TLSVerify        bool
	TLSCACertificate string
	TLSCertificate   string
	TLSKey           string
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

	reader, err := cli.ContainerLogs(
		ctx,
		resp.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			Follow:     true,
		})
	if err != nil {
		log.Fatal(err)
	}

	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
}

func printContainerLogs(cli *client.Client, resp types.ContainerCreateResponse, ctx context.Context) ([]byte, error) {
	out, err := cli.ContainerLogs(
		ctx,
		resp.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})
	if err != nil {
		return nil, err
	}

	defer out.Close()

	content, err := ioutil.ReadAll(out)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// post cluster help types
type helptype int

const (
	Created helptype = iota
	Destroyed
	Updated
)

func clusterHelpError(help helptype, clusterConfigFile string) {
	fmt.Println("Some of the cluster state MAY be available:")
	clusterHelp(help, clusterConfigFile)
}

func clusterHelp(help helptype, clusterConfigFile string) {
	if _, err := os.Stat(path.Join(outputLocation,
		getContainerName(), "admin.kubeconfig")); err == nil {
		fmt.Println("To use kubectl: ")
		fmt.Println(" kubectl --kubeconfig=" + path.Join(
			outputLocation,
			getContainerName(), "admin.kubeconfig") + " [kubectl commands]")
		fmt.Println(" or use 'k2cli tool kubectl --config " + clusterConfigFile + " [kubectl commands]'")

		if _, err := os.Stat(path.Join(outputLocation,
			getContainerName(), "admin.kubeconfig")); err == nil {
			fmt.Println("To use helm: ")
			fmt.Println(" export KUBECONFIG=" + path.Join(
				outputLocation,
				getContainerName(), "admin.kubeconfig"))
			fmt.Println(" helm [helm command] --home " + path.Join(
				outputLocation,
				getContainerName(), ".helm"))
			fmt.Println(" or use 'k2cli tool helm --config " + clusterConfigFile + " [helm commands]'")
		}
	}

	if _, err := os.Stat(path.Join(outputLocation,
		getContainerName(), "ssh_config")); err == nil {
		fmt.Println("To use ssh: ")
		fmt.Println(" ssh <node pool name>-<number> -F " + path.Join(
			outputLocation,
			getContainerName(), "ssh_config"))
		// This is usage has not been implemented. See issue #49
		//fmt.Println(" or use 'k2cli tool --config ssh ssh " + clusterConfigFile + " [ssh commands]'")
	}
}

// Convert dashes to underscore (if any) in cluster name and append to helm_override_ to be able to pull correct env for helm override
func setHelmOverrideEnv(name string) string {
	clusterName := strings.Replace(name, "-", "_", -1)
	helmOverrideVar := "helm_override_" + clusterName
	return helmOverrideVar
}

func containerEnvironment() []string {
	containerName := getContainerName()
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

		deployment := reflect.ValueOf(krakenClusterConfig.Sub("deployment"))
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

// GetDefaultHost produces either the environment-provided host, or a sensible default.
func (conf *DockerClientConfig) GetDefaultHost() string {
	env := os.Getenv("DOCKER_HOST")
	if env == "" {
		return client.DefaultDockerHost
	}
	return env
}

// GetDefaultTLSVerify indicates whether TLS is enabled by the current environment.
func (conf *DockerClientConfig) GetDefaultTLSVerify() bool {
	env := os.Getenv("DOCKER_TLS_VERIFY")
	if (env == "") || (env == "0") {
		return false
	}
	return true
}

// GetDefaultDockerAPIVersion produces either the environment-provided Docker API version, or a sensible default.
func (conf *DockerClientConfig) GetDefaultDockerAPIVersion() string {
	env := os.Getenv("DOCKER_API_VERSION")
	if env == "" {
		return client.DefaultVersion
	}
	return env
}

// GetDefaultTLSCertificatePath produces either the environment-provided path to TLS certificates, or a sensible default.
func (conf *DockerClientConfig) GetDefaultTLSCertificatePath() string {
	env := os.Getenv("DOCKER_CERT_PATH")
	if env == "" {
		return os.ExpandEnv("${HOME}/.docker/")
	}
	return env
}

// GetDefaultTLSCACertificate produces the path to the environment-configured CA certificate for TLS verification.
func (conf *DockerClientConfig) GetDefaultTLSCACertificate() string {
	return filepath.Join(conf.GetDefaultTLSCertificatePath(), "ca.pem")
}

// GetDefaultTLSCertificate produces the path to the environment-configured TLS certificate.
func (conf *DockerClientConfig) GetDefaultTLSCertificate() string {
	return filepath.Join(conf.GetDefaultTLSCertificatePath(), "cert.pem")
}

// GetDefaultTLSKey produces the path to the environment-configured TLS key.
func (conf *DockerClientConfig) GetDefaultTLSKey() string {
	return filepath.Join(conf.GetDefaultTLSCertificatePath(), "key.pem")
}

// Was this config derived solely by OS environment?
// The properties of `conf` will be overridden only by command line args,
// otherwise they're given the values of the associated Default methods.
func (conf *DockerClientConfig) isInheritedFromEnviron() bool {

	boolstr := func(val bool) string {
		if val {
			return "true"
		}
		return "false"
	}

	// This is structured like a table-test, to improve readability. Huzzah!
	compare := map[string][]string{
		"version": []string{conf.DockerAPIVersion, conf.GetDefaultDockerAPIVersion()},
		"host":    []string{conf.DockerHost, conf.GetDefaultHost()},
		"verify":  []string{boolstr(conf.TLSVerify), boolstr(conf.GetDefaultTLSVerify())},
		"cacert":  []string{conf.TLSCACertificate, conf.GetDefaultTLSCACertificate()},
		"cert":    []string{conf.TLSCertificate, conf.GetDefaultTLSCertificate()},
		"key":     []string{conf.TLSKey, conf.GetDefaultTLSKey()},
	}

	for _, vals := range compare {
		if vals[0] != vals[1] {
			return false
		}
	}

	return true

}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
}

func (conf *DockerClientConfig) isTLSActivated() bool {

	components := []bool{ // all of these must be true.
		(conf.TLSEnabled || conf.TLSVerify),
		fileExists(conf.TLSCACertificate),
		fileExists(conf.TLSCertificate),
		fileExists(conf.TLSKey),
	}

	for _, val := range components {
		if !val {
			return false
		}
	}

	return true
}

func getClient() (*client.Client, error) {

	var httpClient *http.Client
	var cli *client.Client
	var err error
	config := dockerClient // global

	if config.isInheritedFromEnviron() {
		// Rely on Docker's own standard environment handling.
		cli, err = client.NewEnvClient()
		if err != nil {
			return nil, err
		}

	} else {
		// Set up an optionally TLS-enabled client, based on non-environment flags.
		// This replicates logic of Docker's `NewEnvClient`, but allows for our
		// command-line argument overrides.
		if config.isTLSActivated() {

			tlsClient, err := tlsconfig.Client(tlsconfig.Options{
				CAFile:             config.TLSCACertificate,
				CertFile:           config.TLSCertificate,
				KeyFile:            config.TLSKey,
				InsecureSkipVerify: !(config.TLSVerify),
			})

			if err != nil {
				return nil, err
			}

			httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsClient,
				},
			}

		}

		headers := map[string]string{
			"User-Agent": fmt.Sprintf("engine-api-cli-%s", config.DockerAPIVersion),
		}

		cli, err = client.NewClient(config.DockerHost, config.DockerAPIVersion, httpClient, headers)
		if err != nil {
			return nil, err
		}
	}

	return cli, nil
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

	base64Auth, err := base64EncodeAuth(authConfig)
	if err != nil {
		return "", err
	}

	return base64Auth, nil
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

	defer pullResponseBody.Close()

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

func containerAction(cli *client.Client, ctx context.Context, command []string, k2config string) (types.ContainerCreateResponse, int, func(), error) {
	var containerResponse types.ContainerCreateResponse

	hostConfig, config_envs := makeMounts(k2config)
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
	clusterName := getContainerName()
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, "k2"+clusterName)
	if err != nil {
		return containerResponse, -1, nil, err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return containerResponse, -1, nil, err
	}

	if verbosity == true {
		backgroundCtx := getContext()
		streamLogs(cli, resp, backgroundCtx)
	}

	statusCode, err := cli.ContainerWait(ctx, resp.ID)
	if err != nil {
		select {
		case <-ctx.Done():
			fmt.Println("Action timed out!")
			return resp, 1, func() {
				// make sure container is killed
				var removeErr error
				if keepAlive {
					removeErr = cli.ContainerKill(
						getContext(),
						resp.ID,
						"KILL")
					if removeErr != nil {
						panic(removeErr)
					}

					newContainerName := "k2-" + namesgenerator.GetRandomName(1)
					removeErr = cli.ContainerRename(
						getContext(),
						resp.ID,
						newContainerName)
					fmt.Println("Renamed k2-" + clusterName + " to " + newContainerName)
				} else {
					removeErr = cli.ContainerRemove(
						getContext(),
						resp.ID,
						types.ContainerRemoveOptions{
							RemoveVolumes: false,
							RemoveLinks:   false,
							Force:         true,
						})
				}
				if removeErr != nil {
					panic(removeErr)
				}
			}, nil
		default:
			return containerResponse, -1, nil, err
		}
	}

	return resp, statusCode, func() {
		var removeErr error
		if keepAlive {
			newContainerName := "k2-" + namesgenerator.GetRandomName(1)
			removeErr = cli.ContainerRename(
				getContext(),
				resp.ID,
				newContainerName)
			fmt.Println("Renamed k2-" + clusterName + " to " + newContainerName)
		} else {
			removeErr = cli.ContainerRemove(
				getContext(),
				resp.ID,
				types.ContainerRemoveOptions{
					RemoveVolumes: false,
					RemoveLinks:   false,
					Force:         false,
				})
		}
		if removeErr != nil {
			panic(removeErr)
		}
	}, nil
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
			if err := ioutil.WriteFile(logFilePath, d, 0644); err == nil {
				os.Remove(logFilePath)
			} else {
				return err
			}

			fileHandle, err = os.Create(logFilePath)
			if err != nil {
				return err
			}
		} else {
			fileHandle, err = os.OpenFile("test.txt", os.O_RDWR, 0666)
		}
	}

	defer fileHandle.Close()

	_, err = fileHandle.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func getContainerName() string {
	// only supports first cluster name right now
	clusters := krakenClusterConfig.Get("deployment.clusters")
	if clusters != nil {
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
