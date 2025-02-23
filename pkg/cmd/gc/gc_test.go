package gc_test

import (
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-test/pkg/cmd/gc"
	"github.com/jenkins-x-plugins/jx-test/pkg/terraforms/tftests"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
)
import (
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	testResources = []string{
		`apiVersion: tf.isaaguilar.com/v1alpha1
kind: Terraform
metadata:
  labels:
    kind: jx-test
    context: myctx
    owner: myowner
    pr: pr-456
    repo: myrepo
  name: tf-myrepo-pr456-myctx-1
  namespace: jx
`,
		`apiVersion: tf.isaaguilar.com/v1alpha1
kind: Terraform
metadata:
  labels:
    kind: jx-test
    context: myctx
    owner: myowner
    pr: pr-456
    repo: myrepo
  name: tf-myrepo-pr456-myctx-2
  namespace: jx
`,
		`apiVersion: tf.isaaguilar.com/v1alpha1
kind: Terraform
metadata:
  labels:
    kind: jx-test
    context: myctx
    owner: myowner
    pr: pr-999
    repo: myrepo
  name: tf-myrepo-pr999-myctx-3
  namespace: jx
`,
	}
)

func TestGC(t *testing.T) {
	ns := "jx"

	useKubectl := true

	scheme := runtime.NewScheme()

	now := time.Now()
	recentTime := now.Add(-1 * time.Hour)
	oldTime := now.Add(-5 * time.Hour)

	fn := func(idx int, u *unstructured.Unstructured) {
		t := oldTime
		if idx > 1 {
			t = recentTime
		}
		u.SetCreationTimestamp(metav1.Time{
			Time: t,
		})
	}

	runner := &fakerunner.FakeRunner{}

	dynObjects := tftests.ParseUnstructureds(t, fn, testResources)
	fakeDynClient := tftests.NewFakeDynClient(scheme, dynObjects...)

	_, o := gc.NewCmdGC()
	o.Namespace = ns
	o.DynamicClient = fakeDynClient
	o.CommandRunner = runner.Run
	o.KubeClient = fake.NewSimpleClientset()

	err := o.Run()
	require.NoError(t, err, "failed to run create command")

	if useKubectl {
		for _, c := range runner.OrderedCommands {
			t.Logf("faked: %s\n", c.CLI())
		}
	} else {
		ctx := o.GetContext()

		list, err := o.Client.List(ctx, metav1.ListOptions{})
		require.NoError(t, err, "failed to list resources")
		require.NotNil(t, list, "no list resource returned")
		require.Len(t, list.Items, 1, "should have GCd resources")

		t.Logf("has remaining Terraform %s\n", list.Items[0].GetName())
	}
}
