package progress

import (
	"fmt"
	"sync"
	"time"
)

type Task struct {
	Index      int
	Name       string
	InProgress bool
	Message    string
	parent     *Progress
	spin       spin
}

type message interface {
	isMessage()
}

type spinNext int

func (sn spinNext) isMessage() {}

type taskAdded Task

func (ta *taskAdded) isMessage() {}

type reflesh int

func (r reflesh) isMessage() {}

type update Task

func (u *update) isMessage() {}

type done Task

func (d *done) isMessage() {}

type halt int

func (r halt) isMessage() {}

type Progress struct {
	tasks     []*Task
	current   int
	message   chan message
	maxLength int
	term      TermManager
	halted    bool
	lock      sync.Mutex
}

func (prog *Progress) drawTask(task *Task) {
	prog.term.Move(0, task.Index)
	prog.term.Write(task.Name)
	prog.term.Move(prog.maxLength+3, task.Index)
	prog.term.Writeln(task.Message)
}

func (prog *Progress) spinNext() {
	for i := 0; i < len(prog.tasks); i++ {
		task := prog.tasks[i]
		if !task.InProgress {
			continue
		}

		prog.term.Move(prog.maxLength+1, i)
		prog.term.Writeln(task.spin.String())
		task.spin.next()
	}
	prog.term.ToEnd()
}

func (prog *Progress) reflesh() {
	for i := 0; i < len(prog.tasks); i++ {
		task := prog.tasks[i]
		prog.term.Move(len(task.Name), i)
		prog.term.EraceRight()
		prog.term.Move(prog.maxLength+1, i)

		if task.InProgress {
			prog.term.Writeln(task.spin.String() + " " + task.Message)
		} else {
			prog.term.Writeln(task.Message)
		}
	}
	prog.term.ToEnd()
}

func (prog *Progress) update(task *Task) {
	prog.term.Move(prog.maxLength+3, task.Index)
	prog.term.EraceRight()
	prog.term.Writeln(task.Message)
	prog.term.ToEnd()
}

func (prog *Progress) done(task *Task) {
	prog.term.Move(len(task.Name), task.Index)
	prog.term.EraceRight()
	prog.term.Move(prog.maxLength+1, task.Index)
	prog.term.Writeln(task.Message)
	prog.term.ToEnd()
}

func (prog *Progress) sentinel() (quit bool) {
	if prog.halted {
		fmt.Println("halted")
		return
	}

	msg := <-prog.message

	switch msg.(type) {
	case spinNext:
		prog.spinNext()
	case *taskAdded:
		prog.drawTask((*Task)(msg.(*taskAdded)))
	case reflesh:
		prog.reflesh()
	case *update:
		prog.update((*Task)(msg.(*update)))
	case *done:
		prog.done((*Task)(msg.(*done)))
	case halt:
		prog.lock.Lock()
		defer prog.lock.Unlock()

		prog.halted = true
		close(prog.message)
		return true
	}

	return false
}

func (prog *Progress) send(msg message) bool {
	prog.lock.Lock()
	defer prog.lock.Unlock()
	if prog.halted {
		return false
	}

	prog.message <- msg
	return true
}

func NewProgress() *Progress {
	prog := &Progress{
		tasks:   make([]*Task, 0),
		message: make(chan message),
	}

	go func() {
		for {
			time.Sleep(50 * time.Millisecond)

			if !prog.send(spinNext(0)) {
				return
			}
		}
	}()

	go func() {
		for {
			if prog.sentinel() {
				return
			}
		}
	}()

	return prog
}

func (prog *Progress) Free() {
	prog.send(halt(0))
	prog.term.ToEnd()
}

func (prog *Progress) NewTask(name, msg string) *Task {
	task := &Task{
		Index:      prog.current,
		Name:       name,
		Message:    msg,
		InProgress: true,
		spin:       spin{},
		parent:     prog,
	}

	prog.current++
	prog.tasks = append(prog.tasks, task)

	if len(name) > prog.maxLength {
		prog.maxLength = len(name)
		prog.send(reflesh(0))
	}

	prog.send((*taskAdded)(task))

	return task
}

func (task *Task) Update(msg string) {
	if !task.InProgress {
		return
	}

	task.Message = msg
	task.parent.send((*update)(task))
}

func (task *Task) Done(msg string) {
	if !task.InProgress {
		return
	}

	task.InProgress = false
	task.Message = msg
	task.parent.send((*done)(task))
}
