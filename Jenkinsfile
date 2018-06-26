pipeline {
    agent {
        kubernetes {
            label 'spot-build'
            yaml """
            apiVersion: v1
            kind: Pod
            metadata:
            labels:
            spec:
              containers:
                - name: jnlp
                  image: hcr.io/jenkins/jnlp-slave:alpine
                - name: helm-kubectl
                  image: dtzar/helm-kubectl
                  command:
                    - cat
                  tty: true
                - name: docker
                  image: docker
                  command:
                    - cat
                  tty: true
                  volumeMounts:
                    - mountPath: /var/run/docker.sock
                      name: docker-sock
              volumes:
                - name: docker-sock
                  hostPath:
                    path: /var/run/docker.sock
                    type: File
            """
        }
    }
    stages {
        stage ("Check Helm Chart") {
            steps {
                container("helm-kubectl") {
                    sh 'helm lint --strict deployments/helm/spot'
                }
            }
        }

        stage ("Build Image") {
            steps {
                container("docker") {
                    sh 'docker build . -t hcr.io/nlowe/spot'
                }
            }
        }

        stage ("Push Image") {
            when {
                branch 'master'
            }
            steps {
                container("docker") {
                    withDockerRegistry([credentialsId: 'hcr-tfsbuild', url: 'https://hcr.io']) {
                        sh 'docker push hcr.io/nlowe/spot'
                    }
                }
            }
        }

        stage ("Deploy") {

            environment {
                KUBECONFIG = credentials('devops-kubeconfig')
                WATCH_JENKINS = credentials('watch-jenkins')
                WATCH_BAMBOO = credentials('watch-bamboo')
                WEBHOOK = credentials('webhook')
            }

            when {
                branch 'master'
            }

            steps {
                container("helm-kubectl") {
                    sh '''
                    export HOME=$PWD

                    kubectl version
                    helm version

                    helm upgrade spot ./deployments/helm/spot/ --install \
                        --namespace spot \
                        --set "watch.jenkins={${WATCH_JENKINS}}" \
                        --set "watch.bamboo={${WATCH_BAMBOO}}" \
                        --set "notify.slack=${WEBHOOK}" \
                        --wait
                    '''
                }
            }
        }
    }
}