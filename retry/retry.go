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

// Retryable errors can be retried if the caller wants
type Retryable interface {
	Retry() error
}

// Check will give a Retryable and true if we can try again
func Check(err error) (Retryable, bool) {
	t, ok := err.(Retryable)
	return t, ok
}

// Error is an error we can retry
type Error struct {
	Message string
	Attempt func() error
}

func (e Error) Error() string {
	return e.Message
}

// Retry makes another attempt
func (e Error) Retry() error {
	return e.Attempt()
}
