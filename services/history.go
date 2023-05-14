/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2023-05-14 14:17
**/

package services

type History struct {
	data []*Pod
}

func NewHistory() *History {
	return &History{data: make([]*Pod, 0)}
}

func (h *History) Set(pod *Pod) {
	h.data = append(h.data, pod)
}

func (h *History) All() []*Pod {
	return h.data
}
