# MinioMatic

MinioMatic is a backend service API for [Minio](https://minio.io/) that provides a simple way to create and manage Minio instances on Kubernetes. It is built using [Golang](https://golang.org/).

## Table of Contents
* [Installation](#installation)
* [Requirements](#requirements)
* [Environment Variables](#environment-variables)
* [API Documentation](#api-documentation)

## Installation

### Docker Image
You can run the application using the Docker image:

Using environment variables file:
```bash
git clone https://github.com/Stenstromen/miniomatic.git
cd miniomatic
cp .env_example .env

# Edit .env file and set the required environment variables

docker run --rm -d \
-p8080:8080 \
--env-file .env \
-v $PWD/assets/:/app/assets:rw \
-v $HOME/.kube/config:/app/config:ro \
ghcr.io/stenstromen/miniomatic:latest

# The API will be available at http://localhost:8080
```

Using discrete environment variables:
```bash
docker run --rm -d \
-p8080:8080 \
-e KUBECONFIG_FILE=/app/config \
-e WILDCARD_DOMAIN=minio.example.com \
-e CLUSTERISSUER=letsencrypt \
-e STORAGECLASSNAME=local-pv \
-e API_KEY=mysecretapikey \
-e ALLOWED_ORIGIN=https://example.com \
-v $PWD/assets/:/app/assets:rw \
-v $HOME/.kube/config:/app/config:ro \
ghcr.io/stenstromen/miniomatic:latest

# The API will be available at http://localhost:8080
```

### Go Binary
You can run the application using the Go binary from the [Releases page](https://github.com/Stenstromen/miniomatic/releases/latest/):

```bash
git clone https://github.com/Stenstromen/miniomatic.git
cd miniomatic
cp .env_example .env
# Edit .env file and set the required environment variables

# Download the binary from the Releases page and place it in the miniomatic directory
tar -xvf miniomatic_*_*.tar.gz 
chmod +x miniomatic

./miniomatic

# The API will be available at http://localhost:8080
```

### Go Build
You can build miniomatic using `go build`:

```bash
git clone https://github.com/Stenstromen/miniomatic.git
cd miniomatic
go build .
cp .env_example .env

# Edit .env file and set the required environment variables

./miniomatic

# The API will be available at http://localhost:8080
```

## Requirements

Before deploying this application, ensure your Kubernetes cluster meets the following prerequisites:

#### 1. Cert-Manager
The application relies on `cert-manager` to automatically provision and manage TLS certificates. Ensure you have `cert-manager` properly installed and configured.

You can follow the [official installation](https://cert-manager.io/docs/installation/kubernetes/) guide to set it up.

#### 2. StorageClass
A specific `StorageClass` is expected for provisioning persistent volumes. Ensure the storage class specified in the `STORAGECLASSNAME` environment variable (default: `local-pv`) is available and properly configured in your cluster.

#### 3. Nginx Ingress Controller
The application utilizes the Nginx Ingress Controller to manage external access to the services in the Kubernetes cluster. Ensure you have the Nginx Ingress Controller set up.

Installation instructions can be found in the [official documentation](https://kubernetes.github.io/ingress-nginx/deploy/).

#### 4. Wildcard Domain
A wildcard domain is required for creating instance-specific subdomains. Ensure you have a wildcard domain configured for your cluster. For example, if you have a domain like `example.com`, you can create a wildcard domain like `*.minio.example.com` to be used for creating instance-specific subdomains.

#### 5. Kubernetes Version
This application has been tested on:
* K3S version: v1.28.2+k3s1

## Environment Variables

#### 1. KUBECONFIG_FILE
* **Description**: Path to your Kubernetes configuration file.
* **Default**: `~/.kube/config`

#### 2. WILDCARD_DOMAIN
* **Description**: The wildcard domain used for creating instance-specific subdomains.
* **Example**: If set to `minio.example.com`, an instance might have a domain like `abc123.minio.example.com`.

#### 3. CLUSTERISSUER
* **Description**: Specifies the cert-manager issuer to be used for generating SSL/TLS certificates for the domains.
* **Default**: `letsencrypt`

#### 4. STORAGECLASSNAME
* **Description**: Specifies the name of the storage class to be used for creating persistent volumes.
* **Default**: `local-pv`
* **Note**: Storage Class needs allowVolumeExpansion set to true in order to be able to resize the volumes.

#### 5. API_KEY
* **Description**: Specifies the API key to be used for authenticating requests to the API.
* **Example**: `mysecretapikey`

#### 6. ALLOWED_ORIGIN
* **Description**: Specifies the origin to be allowed for CORS requests.
* **Example**: `https://example.com`

## API Documentation

### Endpoints

#### 1. Get all instances
* **URL** `/v1/instances`
* **Method** `GET`
* Description: Returns a list of all instances

#### 2. Get single instance
* **URL** `/v1/instances/{id}`
* **Method** `GET`
* Parameters: 
  * `id` - The unique identifier for the instance.
* Description: Returns a single instance

#### 3. Create an instance
* **URL** `/v1/instances`
* **Method** `POST`
* Body:
  * `bucket` - The name of the initial bucket to create.
  * `storage` - The size of the instance in Ki, Mi or Gi (10Gi for example). 
* Description: Creates a new instance and returns its details

#### 4. Update an instance (storage size)
* **URL** `/v1/instances/{id}`
* **Method** `PATCH`
* Parameters: 
  * `id` - The unique identifier for the instance.
* Body:
  * `storage` - The new size of the instance in Ki, Mi or Gi (10Gi for example).
* Description: Updates the storage size of an instance and returns the updated details
* Note: The storage size can only be increased, not decreased. Also, Storage Class needs allowVolumeExpansion set to true in order to be able to resize the volumes

#### 5. Delete an instance
* **URL** `/v1/instances/{id}`
* **Method** `DELETE`
* Parameters: 
  * `id` - The unique identifier for the instance.
* Description: Deletes a specific instance by its ID
