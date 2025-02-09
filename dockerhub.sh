#!/bin/bash

# 检查是否提供了版本号作为参数
if [ -z "$1" ]; then
  echo "Usage: $0 <tag>"
  exit 1
fi

VERSION=$1
IMAGE_NAME="jimxl/mobile_proxy:$VERSION"

# 构建 Docker 镜像
docker build -t $IMAGE_NAME .

# 登录 Docker Hub (如果需要)
# docker login -u your_dockerhub_username -p your_dockerhub_password

# 推送 Docker 镜像到 Docker Hub
docker push $IMAGE_NAME


echo "Successfully built and pushed $IMAGE_NAME to Docker Hub"

# 推送最新标签
LATEST_IMAGE_NAME="jimxl/mobile_proxy:latest"
docker tag $IMAGE_NAME $LATEST_IMAGE_NAME
docker push $LATEST_IMAGE_NAME

echo "Successfully pushed $LATEST_IMAGE_NAME to Docker Hub"
