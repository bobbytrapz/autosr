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

// BoolRetryable errors can be retried if the caller wants
type BoolRetryable interface {
	Retry() (bool, error)
}

// BoolCheck will give a Retryable and true if we can try again
func BoolCheck(err error) (BoolRetryable, bool) {
	t, ok := err.(BoolRetryable)
	return t, ok
}

// BoolError is an error we can retry
type BoolError struct {
	Message string
	Attempt func() (bool, error)
}

func (e BoolError) Error() string {
	return e.Message
}

// Retry makes another attempt
func (e BoolError) Retry() (bool, error) {
	return e.Attempt()
}
