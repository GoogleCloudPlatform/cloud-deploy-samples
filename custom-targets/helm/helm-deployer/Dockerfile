# Copyright 2023 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.21

WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
COPY clouddeploy/go.mod ./clouddeploy/
COPY clouddeploy/*.go ./clouddeploy/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /helm-deployer

#Setup Gcloud (Including gke auth, kubectl and docker-credential-gcr)
RUN curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-440.0.0-linux-x86_64.tar.gz
RUN tar -xf google-cloud-cli-440.0.0-linux-x86_64.tar.gz
RUN google-cloud-sdk/install.sh --quiet --additional-components gke-gcloud-auth-plugin kubectl docker-credential-gcr
RUN ln -s /app/google-cloud-sdk/bin/gcloud /bin/gcloud 
RUN ln -s /app/google-cloud-sdk/bin/gsutil /bin/gsutil 
RUN ln -s /app/google-cloud-sdk/bin/docker-credential-gcloud /bin/docker-credential-gcloud
RUN ln -s /app/google-cloud-sdk/bin/kubectl /bin/kubectl
RUN ln -s /app/google-cloud-sdk/bin/gke-gcloud-auth-plugin /bin/gke-gcloud-auth-plugin

#Setup Helm
RUN curl -O https://get.helm.sh/helm-v3.13.0-linux-amd64.tar.gz
RUN mkdir helm
RUN tar -xf helm-v3.13.0-linux-amd64.tar.gz -C helm
RUN ln -s /app/helm/linux-amd64/helm /bin/helm 

CMD ["/helm-deployer"]