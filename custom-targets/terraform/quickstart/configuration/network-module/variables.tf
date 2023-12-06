/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

variable "network_name" {
    description = "The name of the network being created"
    type = string
}

variable "description" {
    description = "Description of the resource"
    type = string
}

variable "project_id" {
    description = "The ID of the project where the network will be created"
    type = string
}

variable "auto_create_subnetworks" {
    description = "When set to true, the network is created in `auto subnet mode` and it will create a subnet for each region automatically across the 10.128.0.0/9 address range. When set to false, the network is created in `custom subnet mode` so the user can explicitly connect subnetwork resources"
    type = bool
    default = false
}