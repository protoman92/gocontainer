package gomap

import (
	"fmt"
	"reflect"
)

// ChannelConcurrentMap represents a channel-based ConcurrentMap.
type ChannelConcurrentMap interface {
	Map
	Close()
}

type clearRequest struct {
	doneCh chan<- interface{}
}

type containsRequest struct {
	key     interface{}
	foundCh chan<- bool
}

type deleteResult struct {
	prev  interface{}
	found bool
}

type deleteRequest struct {
	key      interface{}
	resultCh chan<- *deleteResult
}

type getResult struct {
	element interface{}
	found   bool
}

type getRequest struct {
	key     interface{}
	valueCh chan<- *getResult
}

type lenRequest struct {
	lenCh chan<- int
}

type keysRequest struct {
	keysCh chan<- []interface{}
}

type setResult struct {
	element interface{}
	found   bool
}

type setRequest struct {
	key   interface{}
	value interface{}
	lenCh chan<- *setResult
}

type stringRequest struct {
	strCh chan<- string
}

type channelConcurrentMap struct {
	storage   Map
	requestCh chan interface{}
}

func (ccm *channelConcurrentMap) Close() {
	close(ccm.requestCh)
}

func (ccm *channelConcurrentMap) String() string {
	strCh := make(chan string, 0)
	ccm.requestCh <- &stringRequest{strCh: strCh}
	return <-strCh
}

// This operation blocks until some result is received.
func (ccm *channelConcurrentMap) Clear() {
	requestCh := make(chan interface{}, 0)
	ccm.requestCh <- &clearRequest{doneCh: requestCh}
	<-requestCh
}

// This operation blocks until a value is received.
func (ccm *channelConcurrentMap) Contains(key interface{}) bool {
	foundCh := make(chan bool, 0)
	ccm.requestCh <- &containsRequest{key: key, foundCh: foundCh}
	return <-foundCh
}

// This operation blocks until some value is received.
func (ccm *channelConcurrentMap) Delete(key interface{}) (interface{}, bool) {
	resultCh := make(chan *deleteResult, 0)
	ccm.requestCh <- &deleteRequest{key: key, resultCh: resultCh}
	result := <-resultCh
	return result.prev, result.found
}

// This operaton blocks until some value is received.
func (ccm *channelConcurrentMap) Get(key interface{}) (interface{}, bool) {
	valueCh := make(chan *getResult, 0)
	ccm.requestCh <- &getRequest{key: key, valueCh: valueCh}
	result := <-valueCh
	return result.element, result.found
}

// This operaton blocks until some value is received.
func (ccm *channelConcurrentMap) Length() int {
	requestCh := make(chan int, 0)
	ccm.requestCh <- &lenRequest{lenCh: requestCh}
	return <-requestCh
}

// This operation blocks untils keys are received.
func (ccm *channelConcurrentMap) Keys() []interface{} {
	keysCh := make(chan []interface{}, 0)
	ccm.requestCh <- &keysRequest{keysCh: keysCh}
	return <-keysCh
}

// This operaton blocks until some value is received.
func (ccm *channelConcurrentMap) Set(key interface{}, value interface{}) (interface{}, bool) {
	lenCh := make(chan *setResult, 0)
	ccm.requestCh <- &setRequest{key: key, value: value, lenCh: lenCh}
	result := <-lenCh
	return result.element, result.found
}

func (ccm *channelConcurrentMap) loopMap() {
	for {
		select {
		case request, ok := <-ccm.requestCh:
			if !ok {
				return
			}

			switch request := request.(type) {
			case *clearRequest:
				ccm.storage.Clear()
				request.doneCh <- true

			case *containsRequest:
				request.foundCh <- ccm.storage.Contains(request.key)

			case *deleteRequest:
				prev, found := ccm.storage.Delete(request.key)
				request.resultCh <- &deleteResult{prev: prev, found: found}

			case *getRequest:
				element, found := ccm.storage.Get(request.key)
				request.valueCh <- &getResult{element: element, found: found}

			case *lenRequest:
				request.lenCh <- ccm.storage.Length()

			case *keysRequest:
				request.keysCh <- ccm.storage.Keys()

			case *setRequest:
				element, found := ccm.storage.Set(request.key, request.value)
				request.lenCh <- &setResult{element: element, found: found}

			case *stringRequest:
				request.strCh <- fmt.Sprint(ccm.storage)

			default:
				panic(fmt.Sprintf("Unrecognized req type %v", reflect.TypeOf(request)))
			}
		}
	}
}

// NewChannelConcurrentMap returns a ChannelConcurrentMap.
func NewChannelConcurrentMap(storage Map) ChannelConcurrentMap {
	cm := &channelConcurrentMap{
		storage:   storage,
		requestCh: make(chan interface{}, 1),
	}

	go cm.loopMap()
	return cm
}
