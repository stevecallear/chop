# see: https://github.com/microsoft/vscode-dev-containers/blob/v0.137.0/containers/go/.devcontainer/base.Dockerfile
ARG VARIANT=1
FROM mcr.microsoft.com/vscode/devcontainers/go:dev-${VARIANT}

# node
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# aws cli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    sudo ./aws/install

# serverless
RUN npm install -g serverless

# mockgen
RUN go get -u -v github.com/golang/mock/mockgen