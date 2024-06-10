


# template-python
Python service template for reuse.

  * [Installation](#installation)
  * [Package](#package)
  * [Docker](#docker)
  * [Helm chart](#helm-chart)
  * [OpenaAPI schema](#openaapi-schema)
  * [Release](#release)
  * [GitHub Actions](#github-actions)
    + [Web service](#web-service)
    + [Library](#library)
  * [Act](#act)
  * [Prometheus metrics](#prometheus-metrics)
  * [Classy-FastAPI](#classy-fastapi)
- [Collaboration guidelines](#collaboration-guidelines)


## Table of Contents

- [template-python](#template-python)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
    - [Development](#development)
    - [Deployment](#deployment)
  - [Development](#development-1)
    - [Step 1: Update and Install Dependencies](#step-1-update-and-install-dependencies)
    - [Step 2: Install Pyenv](#step-2-install-pyenv)
    - [Step 3: Install Python 3.12](#step-3-install-python-312)
    - [Step 4: Connect Poetry to it](#step-4-connect-poetry-to-it)
  - [Docker](#docker)
  - [Manual build and deployment on minikube](#manual-build-and-deployment-on-minikube)
  - [Package](#package)
  - [Helm chart](#helm-chart)
  - [OpenaAPI schema](#openaapi-schema)
  - [Release](#release)
  - [Helm Chart Versioning](#helm-chart-versioning)
  - [GitHub Actions](#github-actions)
    - [Web service](#web-service)
    - [Library](#library)
  - [Act](#act)
  - [Prometheus metrics](#prometheus-metrics)
  - [Classy-FastAPI](#classy-fastapi)
- [Collaboration guidelines](#collaboration-guidelines)

## Prerequisites
### Development
  - [Python 3.12](#step-2-install-pyenv) - look at detailed instructions below
  - [pipx](https://pipx.pypa.io/stable/)
  - [poetry](https://python-poetry.org/docs/)
  - [docker](https://docs.docker.com/get-docker/)
  - [Helm](https://helm.sh/en/docs/)
  - [minikube](https://minikube.sigs.k8s.io/docs/start/)
  - [Act](#act)

### Deployment
  - [Github Actions](#github-actions) - repository use Github Actions to automate the build, test, release and deployment processes. For your convinience we recommend to fill necessary secrets in the repository settings.



## Development
<details>
<h4><summary>Install Python 3.12 if it is not available in your package manager</summary></h4>

These instructions are for Ubuntu 22.04 and may not work for other versions.

Also, these instructions are about using Poetry with Pyenv-managed (non-system) Python.
 
### Step 1: Update and Install Dependencies
Before we install pyenv, we need to update our package lists for upgrades and new package installations. We also need to install dependencies for pyenv. 

Open your terminal and type:  
```bash
sudo apt-get update
sudo apt-get install -y make build-essential libssl-dev zlib1g-dev libbz2-dev \
libreadline-dev libsqlite3-dev wget curl llvm libncursesw5-dev xz-utils \
tk-dev libxml2-dev libxmlsec1-dev libffi-dev liblzma-dev
```

### Step 2: Install Pyenv
We will clone pyenv from the official GitHub repository and add it to our system path.
```bash
git clone https://github.com/pyenv/pyenv.git ~/.pyenv
echo 'export PYENV_ROOT="$HOME/.pyenv"' >> ~/.bashrc
echo 'export PATH="$PYENV_ROOT/bin:$PATH"' >> ~/.bashrc
echo 'eval "$(pyenv init -)"' >> ~/.bashrc
exec "$SHELL"
```
For additional information visit official [docs](https://github.com/pyenv/pyenv?tab=readme-ov-file#installation)

### Step 3: Install Python 3.12
Now that pyenv is installed, we can install different Python versions. To install Python 3.12, use the following command:
```bash
pyenv install 3.12
```

### Step 4: Connect Poetry to it
Do this in the template dir. Pycharm will automatically connect to it later
```bash
poetry env use ~/.pyenv/versions/3.12.1/bin/python
```
(change the version number accordingly to what is installed)

Finally, verify that Poetry indeed is connected to the proper version:
```bash
poetry enf info
```
</details>  


1. If you don't have `Poetry` installed run:
```bash
pipx install poetry
```

2. Install dependencies:
```bash
poetry config virtualenvs.in-project true
poetry install --no-root --with dev,test
```

3. Install `pre-commit` hooks:
```bash
poetry run pre-commit install
```

4. Launch the project:
```bash
poetry run uvicorn app.main:app 
```
or do it in two steps:
```bash
poetry shell
uvicorn app.main:app
```

5. Running tests:
```bash
poetry run pytest
```

You can test the application for multiple versions of Python. To do this, you need to install the required Python versions on your operating system, specify these versions in the `tox.ini` file, and then run the tests:
```bash
poetry run tox
```


## Docker
Build a [Docker](https://docs.docker.com/) image and run a container:
```bash
docker build . -t <image_name>:<image_tag>
docker run <image_name>:<image_tag>
```

Upload the Docker image to the repository:
```bash
docker login -u <username>
docker push <image_name>:<image_tag>
```


## Manual build and deployment on minikube
1. Install [minikube](https://minikube.sigs.k8s.io/docs/start/).
2. Start minikube:
```bash
minikube start
```
3. Build a docker image:
```bash
docker build . -t <image_name>:<image_tag>
```
4. Upload the docker image to minikube:
```bash
minikube image load <image_name>:<image_tag>
```
5. Deploy the service:
```bash
helm upgrade --install <app_name> ./charts/app --set image.repository=<image_name> --set image.tag=latest --version 0.1.0
```

## Package
To generate and publish a package on pypi.org, execute the following commands:
```bash
poetry config pypi-token.pypi <pypi_token>
poetry build
poetry publish
```

pypi_token - API token for authentication on [PyPI](https://pypi.org/help/#apitoken). 


## Helm chart
Authenticate your Helm client in the container registry:
```bash
helm registry login <container_registry> -u <username>
```

Create a [Helm chart](https://helm.sh/docs/):
```bash
helm package charts/<chart_name>
```

Push the Helm chart to container registry:
```bash
helm push <helm_chart_package> <container_registry>
```

Deploy the Helm chart:
```bash
helm repo add <repo_name> <repo_url>
helm repo update <repo_name>
helm upgrade --install <release_name> <repo_name>/<chart_name>
```

## OpenaAPI schema
To manually generate the OpenAPI schema, execute the command:
```bash
poetry run python ./tools/extract_openapi.py app.main:app --app-dir . --out openapi.yaml --app_version <version>
```

## Release
To create a release, add a tag in GIT with the format a.a.a, where 'a' is an integer.
```bash
git tag 0.1.0
git push origin 0.1.0
```
The release version for branches, pull requests, and other tags will be generated based on the last tag of the form a.a.a.

## Helm Chart Versioning
The Helm chart version changed automatically when a new release is created. The version of the Helm chart is equal to the version of the release.

## GitHub Actions
[GitHub Actions](https://docs.github.com/en/actions) triggers testing, builds, and application publishing for each release.  


**Initial setup**  
1. Create the branch [gh-pages](https://pages.github.com/) and use it as a GitHub page.  
2. Set up variables at `https://github.com/<workspace>/<project>/settings/variables/actions`:
- `DOCKER_IMAGE_NAME` - The name of the Docker image for uploading to the repository.
3. Set up secrets at `https://github.com/<workspace>/<project>/settings/secrets/actions`:
- `AWS_ACCESS_KEY_ID` - [AWS Access Key ID.](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html).
- `AWS_SECRET_ACCESS_KEY` - AWS Secret Access Key
- `AWS_REGION` - [AWS region.](https://aws.amazon.com/about-aws/global-infrastructure/regions_az/).
- `EKS_CLUSTER_ROLE_ARN` - The IAM role's ARN in AWS, providing permissions for managing an Amazon EKS Kubernetes cluster.
- `EKS_CLUSTER_NAME` - Amazon EKS Kubernetes cluster name.
- `EKS_CLUSTER_NAMESPACE` - Amazon EKS Kubernetes cluster namespace.
- `HELM_REPO_URL` - `https://<workspace>.github.io/<project>/helm-charts/`


You can set up automatic testing in GitHub Actions for different versions of Python. To do this, specify the versions set in the `.github/workflows/test_and_build.yaml` file. For example:
```yaml
strategy:
  matrix:
    python-version: ["3.10", "3.11", "3.12"]
```

The process of building and publishing differs for web services and libraries.

### Web service
The default build and publish process is configured for a web application (`.github\workflows\build_web.yaml`).
During this process, a Docker image is built, a Helm chart is created, an `openapi.yaml` is generated, and the web service is deployed to a Kubernetes cluster.

**After execution**  
The OpenAPI schema will be available at `https://github.com/<workspace>/<project>/releases/`.  
The index.yaml file containing the list of Helm charts will be available at `https://<workspace>.github.io/<project>/helm-charts/index.yaml`. You can also publish your Helm charts to [Artifact Hub.](https://artifacthub.io/)  
The Docker image will be available at `https://github.com/orgs/<workspace>/packages?repo_name=<project>`.

### Library
To change the build process for the library, you need to replace the nested workflow `./.github/workflows/build_web.yaml` to `./.github/workflows/build_lib.yaml` in `.github/workflows/test_and_build.yaml`:
```yaml
build:
  needs: [test]
  secrets: inherit
  uses: ./.github/workflows/build_lib.yaml
```

After that, during the build process, the package will be built and published on pypi.org.  
Uploading the package to pypi.org only occurs when a.a.a release is created.

**Initial setup**  
Set up a [secret token](https://pypi.org/help/#apitoken) for PyPI at `https://github.com/<workspace>/<project>/settings/secrets/actions`.


**After execution**  
A package will be available at `https://github.com/<workspace>/<project>/releases/` and pypi.org. 

## Act
[Act](https://github.com/nektos/act) allows you to run your GitHub Actions locally (e.g., for developing tests)

Usage example:
```bash
act push -j deploy --secret-file my.secrets
```

## Prometheus metrics
The application includes (`prometheus-fastapi-instrumentator`)[https://github.com/trallnag/prometheus-fastapi-instrumentator] for monitoring performance and analyzing its operation. It automatically adds an endpoint `/metrics` where you can access Prometheus's application metrics. These metrics include information about request counts, request execution times, and other important indicators of application performance.

## Classy-FastAPI
Classy-FastAPI allows you to easily do dependency injection of 
object instances that should persist between FastAPI routes invocations, e.g., database connections.
More on that (with examples) at [Classy-FastAPI GitLab page](https://gitlab.com/companionlabs-opensource/classy-fastapi).

# Collaboration guidelines
HIRO uses and requires from its partners [GitFlow with Forks](https://hirodevops.notion.site/GitFlow-with-Forks-3b737784e4fc40eaa007f04aed49bb2e?pvs=4)
