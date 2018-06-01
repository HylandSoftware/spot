pipeline {
    agent { label 'linux && docker' }
    stages {
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

        stage ("Restart service") {
            when {
                branch 'master'
            }
            steps {
                sh 'echo TODO'
            }
        }
    }
}