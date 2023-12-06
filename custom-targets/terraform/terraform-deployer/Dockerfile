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

ARG GO_VERSION=1.21
ARG TERRAFORM_VERSION=1.2.3
# Used to include the commit SHA in the image.
ARG commit_sha

FROM golang:${GO_VERSION} AS go-build
ENV COMMIT_SHA=$commit_sha
WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X https://github.com/GoogleCloudPlatform/cloud-deploy-samples/custom-targets/util/clouddeploy.GitCommit=$COMMIT_SHA" -o /terraform-deployer

FROM debian:stable-slim AS dependencies
ARG TERRAFORM_VERSION
RUN apt-get update -y && apt-get install unzip wget -y
RUN wget -q https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip

FROM gcr.io/distroless/static-debian12:latest AS release
COPY --from=go-build /terraform-deployer /bin/terraform-deployer
COPY --from=dependencies /terraform /bin/terraform

CMD ["/bin/terraform-deployer"]