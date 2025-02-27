package channel

import (
	"fmt"
	"regexp"
	"time"

	"github.com/scrapli/scrapligo/util"
)

// SendInputB sends the given input bytes to the device and returns the bytes read.
func (c *Channel) SendInputB(input []byte, opts ...util.Option) ([]byte, error) {
	c.l.Debugf("channel SendInput requested, sending input '%s'", input)

	op, err := NewOperation(opts...)
	if err != nil {
		return nil, err
	}

	cr := make(chan *result)

	go func() {
		var b []byte

		err = c.Write(input, false)
		if err != nil {
			cr <- &result{b: b, err: err}

			return
		}

		_, err = c.ReadUntilInput(input)
		if err != nil {
			cr <- &result{b: b, err: err}

			return
		}

		err = c.WriteReturn()
		if err != nil {
			cr <- &result{b: b, err: err}

			return
		}

		if !op.Eager {
			var nb []byte

			var readErr error

			if len(op.InterimPromptPatterns) == 0 {
				nb, readErr = c.ReadUntilPrompt()
			} else {
				prompts := []*regexp.Regexp{c.PromptPattern}
				prompts = append(prompts, op.InterimPromptPatterns...)

				nb, readErr = c.ReadUntilAnyPrompt(prompts)
			}

			if readErr != nil {
				cr <- &result{b: b, err: readErr}
			}

			b = append(b, nb...)
		}

		cr <- &result{
			b:   c.processOut(b, op.StripPrompt),
			err: nil,
		}
	}()

	timer := time.NewTimer(c.GetTimeout(op.Timeout))

	select {
	case r := <-cr:
		if r.err != nil {
			return nil, r.err
		}

		return r.b, nil
	case <-timer.C:
		c.l.Critical("channel timeout sending input to device")

		return nil, fmt.Errorf("%w: channel timeout sending input to device", util.ErrTimeoutError)
	}
}

// SendInput sends the input string to the target device. Any bytes output is returned.
func (c *Channel) SendInput(input string, opts ...util.Option) ([]byte, error) {
	return c.SendInputB([]byte(input), opts...)
}
