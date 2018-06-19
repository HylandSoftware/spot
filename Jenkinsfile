pipeline {
    agent { label 'linux && docker' }
    stages {
        stage ("Check Helm Chart") {
            agent {
                docker {
                    image 'dtzar/helm-kubectl'
                    label 'linux && docker'
                    reuseNode true
                }
            }

            steps {
                sh 'helm lint --strict deployments/helm/spot'
            }
        }

        stage ("Build Image") {
            steps {
                sh 'docker build . -t hcr.io/nlowe/spot'
            }
        }

        stage ("Push Image") {
            when {
                branch 'master'
            }
            steps {
                withDockerRegistry([credentialsId: 'hcr-tfsbuild', url: 'https://hcr.io']) {
                    sh 'docker push hcr.io/nlowe/spot'
                }
            }
        }

        stage ("Deploy") {
            agent {
                docker {
                    image 'dtzar/helm-kubectl'
                    label 'linux && docker'
                    reuseNode true
                }
            }

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