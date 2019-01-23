package link

/*/ #cgo CFLAGS: -O3 */

type Line_t struct {
	Start int64
	End   int64
}

type TaskNode struct {
	Line_t
	Next *TaskNode
}

type TaskLink struct {
	length int
	header TaskNode
}

/**
 * start: 4
 * length: 3
 * end: start + length = 7
 * content: [4, 5, 6]
 */

func line(foo_s int64, foo_e int64, bar_s int64, bar_e int64) (vec int, start int64, end int64) {
	// foo在左边
	if foo_e < bar_s {
		return -1, -1, -1
	}
	// foo在右边
	if bar_e < foo_s {
		return 1, -1, -1
	}
	tar_s := foo_s
	// foo的起点在bar中间
	if bar_s < foo_s && foo_s < bar_e {
		tar_s = bar_s
	}
	tar_e := foo_e
	if bar_s < foo_e && foo_e < bar_e {
		tar_e = bar_e
	}
	return 0, tar_s, tar_e
}

func _NewTaskLink() *TaskLink {
	this := &TaskLink{
		length: 0,
		header: TaskNode {
			Next: nil,
		},
	}

	return this
}

func New(arr []Line_t) *TaskLink {
	this := _NewTaskLink()
	if nil == arr {
		return this
	}

	this.length = len(arr)
	p := &(this.header)

	for i := 0; i < this.length; i++ {
		newNode := &TaskNode {
			Next: nil,
		}
		newNode.Start = arr[i].Start
		newNode.End = arr[i].End
		p.Next = newNode
		p = newNode
	}

	return this
}

// 挂载
func (this *TaskLink) Mount(start int64, end int64) {
	newNode := &TaskNode {
		Line_t: Line_t {
			Start: start,
			End: end,
		},
		Next: nil,
	}

	p := &(this.header)

	for {
		// 尽头
		if nil == p.Next {
			p.Next = newNode
			this.length += 1
			return
		}
		curNode := p.Next

		vec, lineStart, lineEnd := line(start, end, curNode.Start, curNode.End)

		switch vec {
		case -1:
			// 头插入
			newNode.Next = curNode
			p.Next = newNode
			curNode = newNode
			this.length += 1
		case 0:
			// 节点扩大
			curNode.Start = lineStart
			curNode.End = lineEnd
		default:
			// 下一个节点
			p = p.Next
			continue
		}

		// 如果前序不是头节点，与前序节点连接
		if p != &(this.header) && curNode.Start <= p.End {
			p.End = curNode.End
			p.Next = curNode.Next
			this.length -= 1
		}
		return
	}
}

// 转成数组
func (this *TaskLink) ToArray() []Line_t {
	ret := make([]Line_t, this.length)
	p := this.header.Next

	for i := 0; i < this.length; i++ {
		ret[i].Start = p.Start
		ret[i].End = p.End
		p = p.Next
	}
	return ret
}
