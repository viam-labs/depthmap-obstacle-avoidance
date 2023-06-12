package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/geo/r3"

	"github.com/edaniels/golog"
	"github.com/edaniels/gostream"
	"go.viam.com/rdk/components/base"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/rimage"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/utils"
	"go.viam.com/utils/rpc"
)

const (
	roverSpeed = 1000.0
	thresh     = 500
	camName    = "intel:depth"
	baseName   = "viam_base"
)

func main() {
	// Connect to the robot
	logger := golog.NewDevelopmentLogger("client")
	robot, err := client.New(
		context.Background(),
		// Replace "blah.viam.cloud" with your URL
		"blah.viam.cloud",
		logger,
		client.WithDialOptions(rpc.WithCredentials(rpc.Credentials{
			Type: utils.CredentialsTypeRobotLocationSecret,
			// Replace "<SECRET>" (including brackets) with your robot's secret
			Payload: "<SECRET>",
		})),
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer robot.Close(context.Background())

	// Print available resources
	logger.Info("Resources:")
	logger.Info(robot.ResourceNames())

	// Grab the camera component
	camComponent, err := camera.FromRobot(robot, camName)
	if err != nil {
		logger.Error(err)
	}

	// Grab the base component
	roverBase, err := base.FromRobot(robot, baseName)
	if err != nil {
		logger.Fatalf("cannot get base: %v", err)
	}

	// Set up threshold and obstacle channel
	medianThresh := rimage.Depth(thresh)
	hasObstacle := make(chan bool)

	// Populate obstacle channel with true or false
	go func() {
		checkForObstacle(context.Background(), camComponent, medianThresh, hasObstacle)
	}()

	// Move base according to obstacle channel (true or false)
	go func() {
		moveBase(context.Background(), roverBase, hasObstacle)
	}()

	// Print the results as fast as they come in
	for {
		fmt.Println(<-hasObstacle)
	}
}

// checkForObstacle uses the input depth camera to stream depth images, check the median depth against
// the input threshold, and update the output channel with a boolean (true when median < threshold)
func checkForObstacle(ctx context.Context, cam camera.Camera, depThresh rimage.Depth, out chan<- bool) {
	// Setup stream with request for depth images
	ctx = gostream.WithMIMETypeHint(ctx, utils.MimeTypeRawDepth)
	camStream, err := cam.Stream(ctx)
	if err != nil {
		fmt.Println(err)
	}
	// Grab a depth image from the stream and determine if there's an obstacle
	for {
		pic, release, err := camStream.Next(ctx)
		if err != nil {
			continue
		}
		defer release()

		// Get the data from the depth map
		dm, err := rimage.ConvertImageToDepthMap(ctx, pic)
		if err != nil {
			fmt.Println(err)
		}
		depData := dm.Data()

		// Sort the depth data [smallest...largest]
		sort.Slice(depData, func(i, j int) bool {
			return depData[i] < depData[j]
		})
		med := int(0.5 * float64(len(depData)))

		// Check median value against threshold
		if depData[med] < depThresh {
			out <- true
		} else {
			out <- false
		}
	}
}

// moveBase provides a loop in which we drive straight as long as there are not obstacles
// when an obstacle is detected, we stop, spin a bit, and go again.
func moveBase(ctx context.Context, base base.Base, obstacle chan bool) {
	backUp := 300
	for {
		if <-obstacle {
			base.Stop(ctx, nil)
			base.MoveStraight(ctx, -backUp, roverSpeed, nil)
			base.Spin(ctx, 120, 360, nil)
		} else {
			base.SetVelocity(ctx, r3.Vector{Y: roverSpeed}, r3.Vector{Z: 0}, nil)
		}
	}
}
