/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2023-05-14 14:09
**/

package services

import hash "github.com/lemonyxk/structure/map"

type Pods struct {
	data *hash.Hash[string, *Pod]
}

func NewPods() *Pods {
	return &Pods{data: hash.New[string, *Pod]()}
}

func (p *Pods) Set(name string, pod *Pod) {
	p.data.Set(name, pod)
}

func (p *Pods) Get(name string) *Pod {
	return p.data.Get(name)
}

func (p *Pods) Range(fn func(name string, pod *Pod) bool) {
	p.data.Range(func(k string, v *Pod) bool {
		return fn(k, v)
	})
}

func (p *Pods) Len() int {
	return p.data.Len()
}

func (p *Pods) Delete(name string) {
	p.data.Delete(name)
}
