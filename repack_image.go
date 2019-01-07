package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/image/image"
	"github.com/containers/image/pkg/blobinfocache"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/pkg/errors"
)

// Converts container image to Lambda layer archive files
func RepackImage(imageName string, layerOutputDir string) (retErr error) {
	// Get image's layer data from image name
	ref, err := alltransports.ParseImageName(imageName)
	if err != nil {
		return err
	}

	sys := &types.SystemContext{}

	ctx := context.Background()

	cache := blobinfocache.DefaultCache(sys)

	rawSource, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return err
	}

	src, err := image.FromSource(ctx, sys, rawSource)
	if err != nil {
		if closeErr := rawSource.Close(); closeErr != nil {
			return errors.Wrapf(err, " (close error: %v)", closeErr)
		}

		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			retErr = errors.Wrapf(retErr, " (close error: %v)", err)
		}
	}()

	layerInfos := src.LayerInfos()

	// Unpack and inspect each image layer, copy relevant files to new Lambda layer
	if err := os.MkdirAll(layerOutputDir, 0777); err != nil {
		return err
	}

	lambdaLayerNum := 1

	for _, layerInfo := range layerInfos {
		lambdaLayerFilename := filepath.Join(layerOutputDir, fmt.Sprintf("layer-%d.zip", lambdaLayerNum))

		layerStream, _, err := rawSource.GetBlob(ctx, layerInfo, cache)
		if err != nil {
			return err
		}
		defer layerStream.Close()

		fileCreated, err := RepackLayer(lambdaLayerFilename, layerStream)
		if err != nil {
			return err
		}

		if fileCreated {
			lambdaLayerNum++
		}
	}

	return nil
}
