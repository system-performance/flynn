package main

import (
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/tarreceive/utils"
	c "github.com/flynn/go-check"
)

type TarreceiveSuite struct {
	Helper
}

var _ = c.ConcurrentSuite(&TarreceiveSuite{})

// TestConvertWhitouts ensures that AUFS whiteouts are converted to OverlayFS
// whiteouts and have the same effect (i.e. hiding removed files)
func (s *TarreceiveSuite) TestConvertWhiteouts(t *c.C) {
	// build a Docker image with whiteouts
	repo := "tarreceive-test-whiteouts"
	s.buildDockerImage(t, repo,
		"RUN echo foo > /foo.txt",
		"RUN rm /foo.txt",
		"RUN mkdir /opaque && touch /opaque/file.txt",
		"RUN rm -rf /opaque && mkdir /opaque",
	)

	// create app
	client := s.controllerClient(t)
	app := &ct.App{Name: repo}
	t.Assert(client.CreateApp(app), c.IsNil)

	// flynn docker push image
	t.Assert(flynn(t, "/", "-a", app.Name, "docker", "push", repo), Succeeds)

	// check the whiteouts are effective
	t.Assert(flynn(t, "/", "-a", app.Name, "run", "sh", "-c", "[[ ! -f /foo.txt ]]"), Succeeds)
	t.Assert(flynn(t, "/", "-a", app.Name, "run", "sh", "-c", "[[ ! -f /opaque/file.txt ]]"), Succeeds)
}

// TestReleaseDeleteImageLayers ensures that deleting a release which uses an
// image which has shared layers does not delete the shared layers
func (s *TarreceiveSuite) TestReleaseDeleteImageLayers(t *c.C) {
	// build Docker images with shared layers and push them to two
	// different apps
	app1 := "tarreceive-test-delete-layers-1"
	s.buildDockerImage(t, app1,
		"RUN echo shared-layer > /shared.txt",
		"RUN echo app1-layer > /app1.txt",
	)
	t.Assert(flynn(t, "/", "create", "--remote", "", app1), Succeeds)
	t.Assert(flynn(t, "/", "-a", app1, "docker", "push", app1), Succeeds)

	app2 := "tarreceive-test-delete-layers-2"
	s.buildDockerImage(t, app2,
		"RUN echo shared-layer > /shared.txt",
		"RUN echo app2-layer > /app2.txt",
	)
	t.Assert(flynn(t, "/", "create", "--remote", "", app2), Succeeds)
	t.Assert(flynn(t, "/", "-a", app2, "docker", "push", app2), Succeeds)

	// get the two images
	client := s.controllerClient(t)
	release1, err := client.GetAppRelease(app1)
	t.Assert(err, c.IsNil)
	image1, err := client.GetArtifact(release1.ArtifactIDs[0])
	t.Assert(err, c.IsNil)
	release2, err := client.GetAppRelease(app2)
	t.Assert(err, c.IsNil)
	image2, err := client.GetArtifact(release2.ArtifactIDs[0])
	t.Assert(err, c.IsNil)

	// check that the two apps have some common image layers but different
	// artifacts
	image1Layers := make(map[string]struct{}, len(image1.Manifest().Rootfs[0].Layers))
	for _, layer := range image1.Manifest().Rootfs[0].Layers {
		image1Layers[layer.ID] = struct{}{}
	}
	image2Layers := make(map[string]struct{}, len(image2.Manifest().Rootfs[0].Layers))
	for _, layer := range image2.Manifest().Rootfs[0].Layers {
		image2Layers[layer.ID] = struct{}{}
	}
	commonLayers := make(map[string]struct{})
	distinctLayers := make(map[string]struct{})
	for id := range image1Layers {
		if _, ok := image2Layers[id]; ok {
			commonLayers[id] = struct{}{}
		} else {
			distinctLayers[id] = struct{}{}
		}
	}
	t.Assert(commonLayers, c.Not(c.HasLen), 0)
	t.Assert(distinctLayers, c.Not(c.HasLen), 0)
	t.Assert(image1.ID, c.Not(c.Equals), image2.ID)

	// check all the layers exist at the paths we expect in the blobstore
	getLayer := func(id string) *CmdResult {
		url := utils.LayerURL(id)
		return flynn(t, "/", "-a", "blobstore", "run", "curl", "-fsSLo", "/dev/null", "--write-out", "%{http_code}", url)
	}
	assertExist := func(layers map[string]struct{}) {
		for id := range layers {
			res := getLayer(id)
			t.Assert(res, Succeeds)
			t.Assert(res, Outputs, "200")
		}
	}
	assertNotExist := func(layers map[string]struct{}) {
		for id := range layers {
			res := getLayer(id)
			t.Assert(res, c.Not(Succeeds))
			t.Assert(res, OutputContains, "404 Not Found")
		}
	}
	assertExist(commonLayers)
	assertExist(distinctLayers)

	// delete app1 and check the distinct layers were deleted but the
	// common layers still exist
	t.Assert(flynn(t, "/", "-a", app1, "delete", "--yes"), Succeeds)
	assertNotExist(distinctLayers)
	assertExist(commonLayers)

	// delete app2 and check we can push app1's image to a new app and have
	// the layers regenerated (which checks tarreceive cache invalidation)
	t.Assert(flynn(t, "/", "-a", app2, "delete", "--yes"), Succeeds)
	app3 := "tarreceive-test-delete-layers-3"
	t.Assert(flynn(t, "/", "create", "--remote", "", app3), Succeeds)
	t.Assert(flynn(t, "/", "-a", app3, "docker", "push", app1), Succeeds)
	t.Assert(flynn(t, "/", "-a", app3, "run", "test", "-f", "/app1.txt"), Succeeds)
}

// TestTabsInEnv ensures that a docker container containing tabs
// in the environment variables can be imported.
func (s *TarreceiveSuite) TestTabsInEnv(t *c.C) {
	// build a Docker image with tabs in env
	repo := "tarreceive-test-tab-env"
	s.buildDockerImage(t, repo,
		"ENV TAB test\ttest",
	)

	// create app
	client := s.controllerClient(t)
	app := &ct.App{Name: repo}
	t.Assert(client.CreateApp(app), c.IsNil)

	// flynn docker push image
	t.Assert(flynn(t, "/", "-a", app.Name, "docker", "push", repo), Succeeds)

	// check the environment variable has the correct value
	t.Assert(flynn(t, "/", "-a", app.Name, "run", "sh", "-c", "[[ \"$TAB\" = \"test\ttest\" ]]"), Succeeds)
}
