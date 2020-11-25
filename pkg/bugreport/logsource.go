package bugreport

import (
	"io"
	"path"

	v1 "k8s.io/api/core/v1"
)

type logSource struct {
	ns   v1.Namespace
	pod  v1.Pod
	cont v1.Container
}

func (l *logSource) pathName() string {
	return path.Join(Namespace, l.ns.Name, l.pod.Name, l.cont.Name)
}

func (l *logSource) fetchRcForLogSource(cs coreClient) (io.ReadCloser, error) {
	options := v1.PodLogOptions{Timestamps: true, Container: l.cont.Name}
	return cs.CoreV1().Pods(l.ns.Name).GetLogs(l.pod.Name, &options).Stream()
}
