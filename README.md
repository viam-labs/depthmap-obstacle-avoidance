# depthmap-obstacle-avoidance

## In this repo... ##
This repository contains code for performing obstacle detection using only a depth camera.  Specifically, the script main.go is meant for use with the Viam RDK (ideally a Viam rover!) and an Intel RealSense camera.  See [here](https://github.com/viamrobotics/camera-servers) for an explanation of how to set up your Intel RealSense camera.

Some configurable parameters (like the names of the components) are saved as constant variables.  Also, be sure to replace the URL and robot secret in the code with your actual robot's URL and secret.