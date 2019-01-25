// This file is part of autosr.
//
// autosr is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// autosr is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with autosr.  If not, see <https://www.gnu.org/licenses/>.

package retry

import (
	"time"
)

// TimeRetryable errors can be retried if the caller wants
type TimeRetryable interface {
	Retry() (string, error)
}

// TimeCheck will give a Retryable and true if we can try again
func TimeCheck(err error) (TimeRetryable, bool) {
	t, ok := err.(TimeRetryable)
	return t, ok
}

// TimeError is an error we can retry
type TimeError struct {
	Message string
	Attempt func() (time.Time, error)
}

func (e TimeError) Error() string {
	return e.Message
}

// Retry makes another attempt
func (e TimeError) Retry() (time.Time, error) {
	return e.Attempt()
}
