// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

package itertools

func Map[S ~[]E, E, R any](x S, f func(E) R) []R {
	results := make([]R, 0, len(x))
	for _, e := range x {
		results = append(results, f(e))
	}
	return results
}
