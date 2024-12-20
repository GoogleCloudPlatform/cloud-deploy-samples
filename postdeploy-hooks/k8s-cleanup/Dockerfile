# Copyright 2024 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
ARG GCLOUD_VERSION=501.0.0
ARG GO_VERSION=1.21

FROM golang:${GO_VERSION} AS build
WORKDIR /cleanup
COPY go.mod go.sum *.go ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /cleanup-kubernetes-resources
# Download kubectl
RUN curl -L "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" > kubectl \
    && chmod +x kubectl \
    && mv kubectl /usr/bin/kubectl

FROM  gcr.io/google.com/cloudsdktool/google-cloud-cli:${GCLOUD_VERSION}
WORKDIR /
COPY --from=build /cleanup-kubernetes-resources /cleanup-kubernetes-resources
COPY --from=build /usr/bin/kubectl /usr/bin/kubectl
CMD ["/cleanup-kubernetes-resources"]