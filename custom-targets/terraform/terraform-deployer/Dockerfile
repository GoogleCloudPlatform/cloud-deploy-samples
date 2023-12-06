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

FROM golang:1.21 as go-build
WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /terraform-deployer

FROM debian:stable-slim AS release
RUN apt-get update -y && apt-get install unzip wget -y

# Setup Terraform.
ARG TERRAFORM_VERSION=1.2.3
WORKDIR /
RUN wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN ln -s /terraform /bin/terraform
RUN terraform --version 
COPY --from=go-build /terraform-deployer /terraform-deployer

CMD ["/terraform-deployer"]