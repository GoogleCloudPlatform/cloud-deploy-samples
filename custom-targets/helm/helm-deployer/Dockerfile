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

ARG GCLOUD_VERSION=456.0.0
ARG GO_VERSION=1.21
ARG HELM_VERSION=3.13.2

FROM golang:${GO_VERSION} AS go-build
ARG COMMIT_SHA=unknown
WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=${COMMIT_SHA}" -o /helm-deployer

FROM debian:stable-slim AS dependencies
ARG HELM_VERSION
RUN apt-get update -y && apt-get install unzip wget -y
RUN wget -q https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz
RUN tar xvzf helm-v${HELM_VERSION}-linux-amd64.tar.gz linux-amd64/helm

FROM  gcr.io/google.com/cloudsdktool/google-cloud-cli:${GCLOUD_VERSION} AS release
COPY --from=go-build /helm-deployer /bin/helm-deployer
COPY --from=dependencies /linux-amd64/helm /bin/helm

CMD ["/bin/helm-deployer"]
