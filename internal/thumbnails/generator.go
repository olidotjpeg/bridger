package thumbnails

import (
	"container/list"
	"fmt"
	"log"
)

type Thumbnail struct{}

type Queue struct {
	data *list.List
}

type QueueAsset struct {
	assetID string
	path    string
}

func NewQueue() *Queue {
	return &Queue{data: list.New()}
}

func (q *Queue) Enqueue(assetID, path string) {
	log.Printf("ASSETID %s", assetID)
	log.Printf("PATH %s", path)
	// asset := QueueAsset{assetID: assetID, path: path}
	// q.data.PushBack(asset)
}

func (q *Queue) Dequeue() (int, error) {
	if q.IsEmpty() {
		return 0, fmt.Errorf("queue is empty")
	}
	front := q.data.Front()
	q.data.Remove(front)
	return front.Value.(int), nil
}

func (q *Queue) Front() (int, error) {
	if q.IsEmpty() {
		return 0, fmt.Errorf("queue is empty")
	}
	return q.data.Front().Value.(int), nil
}

func (q *Queue) IsEmpty() bool {
	return q.data.Len() == 0
}

func (q *Queue) Size() int {
	return q.data.Len()
}

func (l *Thumbnail) Generate(assetID, path string) {
	fmt.Printf("ALOHA")
}
