FROM hitzhangjie/gitpod:latest

RUN sudo apt-get update \
 && sudo apt-get install -y \
    graphviz \
 && sudo rm -rf /var/lib/apt/lists/*
