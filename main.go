package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

type BMBuildkit struct {
}

// example usage: "dagger call build --device-spec lattice/ice40/yosys --target dciangot/my_fpga_firmware:v1 --context ./examples/blinky/ice40 "
func (m *BMBuildkit) Build(ctx context.Context, deviceSpec string, target string, contextDir *Directory, imageRef Optional[string], push Optional[bool], appendManifest Optional[bool]) (string, error) {

	//docker run -e MODULE_NAME=blinky -e SYNTH_FILE=blinky.v -v $PWD/examples/blinky:/opt/source -ti dciangot/yosys bash

	pushing := push.GetOr(false)
	if pushing {
		_, err := dag.Container().
			From("dciangot/yosys:latest").
			WithMountedDirectory("/opt/source", contextDir).
			WithWorkdir("/opt/context").
			WithExec([]string{"ls", "-altrh"}).
			Stdout(ctx)
		if err != nil {
			return "", err
		}

		return m.Push(ctx, contextDir.File("firmware.bin"), deviceSpec, target, contextDir, appendManifest)
	}

	return dag.Container().
		From("dciangot/yosys:latest").
		WithMountedDirectory("/opt/source", contextDir).
		WithWorkdir("/opt/context").
		WithExec([]string{"ls", "-altrh"}).
		Stdout(ctx)
}

// example usage: "dagger call push --target dciangot/my_fpga_firmware:v1 --firmware ./examples/blinky/ice40/firmware.bin --bring-context ./examples/blinky/ice40 --device-spec lattice/ice40/yosys"
func (m *BMBuildkit) Push(ctx context.Context, firmware *File, deviceSpec string, target string, bringContext *Directory, appendManifest Optional[bool]) (string, error) {
	var platforms = []Platform{
		Platform(deviceSpec),
		Platform("bm/context"),
	}
	//shouldIAppend := appendManifest.GetOr(false)
	jsonBytes, err := crane.Manifest(target)
	if err != nil {

		platformVariants := make([]*Container, 0, len(platforms))

		for _, platform := range platforms {
			if platform == "bm/context" {
				ctr := dag.Container(ContainerOpts{Platform: platform}).
					//WithLabel("org.opencontainers.image.lattice.ice40", "").
					WithDirectory("/firmware.bin", bringContext)
				platformVariants = append(platformVariants, ctr)
			} else {
				ctr := dag.Container(ContainerOpts{Platform: platform}).
					//WithLabel("org.opencontainers.image.lattice.ice40", "").
					WithFile("/firmware.bin", firmware)
				platformVariants = append(platformVariants, ctr)
			}

		}

		return dag.
			Container().
			Publish(ctx, target, ContainerPublishOpts{
				PlatformVariants: platformVariants,
				// Some registries may require explicit use of docker mediatypes
				// rather than the default OCI mediatypes
				// MediaTypes: dagger.Dockermediatypes,
			})
	} else {
		index, err := cranev1.ParseIndexManifest(bytes.NewReader(jsonBytes))
		if err != nil {
			return "", err
		}

		var platformVariants []*Container
		var platform Platform
		var manlist []string

		isPlatformAlready := false

		for _, manifest := range index.Manifests {
			manlist = append(manlist, manifest.Platform.String())
			if manifest.Platform.String() == deviceSpec {
				isPlatformAlready = true
				platform = Platform(manifest.Platform.String())
				ctr := dag.Container(ContainerOpts{Platform: platform}).
					//WithLabel("org.opencontainers.image.lattice.ice40", "").
					WithFile("/firmware.bin", firmware)
				platformVariants = append(platformVariants, ctr)
			} else {
				platform = Platform(manifest.Platform.String())
				ctr := dag.Container(ContainerOpts{Platform: platform}).From(target)
				platformVariants = append(platformVariants, ctr)
			}
		}

		if !isPlatformAlready {
			manlist = append(manlist, deviceSpec)
			platform = Platform(deviceSpec)
			ctr := dag.Container(ContainerOpts{Platform: platform}).
				//WithLabel("org.opencontainers.image.lattice.ice40", "").
				WithFile("/firmware.bin", firmware)
			platformVariants = append(platformVariants, ctr)
		}

		dag.
			Container().
			Publish(ctx, target, ContainerPublishOpts{
				PlatformVariants: platformVariants,
				// Some registries may require explicit use of docker mediatypes
				// rather than the default OCI mediatypes
				// MediaTypes: dagger.Dockermediatypes,
			})

		return fmt.Sprintln(manlist), nil

	}
}

// DumpContext

// DumpFirmware

// LoadFirmware