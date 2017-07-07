podTemplate(label: 'k2cli', containers: [
    containerTemplate(name: 'jnlp', image: 'quay.io/samsung_cnct/custom-jnlp:0.1', args: '${computer.jnlpmac} ${computer.name}'),
    containerTemplate(name: 'golang', image: 'golang:latest', ttyEnabled: true, command: 'cat', alwaysPullImage: true, resourceRequestMemory: '1Gi', resourceLimitMemory: '1Gi'),
    containerTemplate(name: 'k2-tools', image: 'quay.io/samsung_cnct/k2-tools:latest', ttyEnabled: true, command: 'cat', alwaysPullImage: true, resourceRequestMemory: '1Gi', resourceLimitMemory: '1Gi')
    ], volumes: [
      hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock')//,
      // hostPathVolume(hostPath: '/var/lib/docker/scratch', mountPath: '/mnt/scratch'),
      // secretVolume(mountPath: '/home/jenkins/.docker/', secretName: 'samsung-cnct-quay-robot-dockercfg')
    ]) {
        node('k2cli') {
            customContainer('golang') {

                stage('hello!') {
                    echo 'hello world!'
                }

                stage('checkout') {
                    checkout scm
                    kubesh 'go version'
                }

                stage('build') {
                    kubesh 'go get -v -d -t ./... || true'
                    kubesh 'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ${HOME}/k2cli'
                }


                stage('aws config generation') {
                    kubesh '${HOME}/k2cli -x -v generate /tmp/config.yaml'
                }

                stage ('find home') {
                    kubesh 'find /tmp'
                }

                stage('cat config file') {
                    kubesh 'cat /tmp/config.yaml'
                }

                stage('update generated aws config') {
                    kubesh "build-scripts/update-generated-config.sh /tmp/config.yaml ${env.JOB_BASE_NAME}-${env.BUILD_ID}"
                }

                stage("read config file again") {
                    kubesh 'cat /tmp/config.yaml'
                }



            }
            customContainer('k2-tools'){

                stage('checkout') {
                    checkout scm
                }
                stage('fetch credentials') {
                    kubesh 'build-scripts/fetch-credentials.sh'
                    kubesh 'ls -R'
                }




            }


        }
    }
def kubesh(command) {
  if (env.CONTAINER_NAME) {
    if ((command instanceof String) || (command instanceof GString)) {
      command = kubectl(command)
    }

    if (command instanceof LinkedHashMap) {
      command["script"] = kubectl(command["script"])
    }
  }

  sh(command)
}

def kubectl(command) {
  "kubectl exec -i ${env.HOSTNAME} -c ${env.CONTAINER_NAME} -- /bin/sh -c 'cd ${env.WORKSPACE} && ${command}'"
}

def customContainer(String name, Closure body) {
  withEnv(["CONTAINER_NAME=$name"]) {
    body()
  }
}


// vi: ft=groovy
