package runtime

// topoSortServices orders ids so each service comes after everything in its
// DependsOn list, restricted to ids within the same set — a dependency
// outside the set doesn't constrain ordering here since it isn't something
// this call is starting/stopping. If a cycle is found among ids, it returns
// the original input order unchanged and hadCycle=true (T-051 MVP fallback:
// config order + a logged warning from the caller).
func topoSortServices(ids []string, dependsOn map[string][]string) (order []string, hadCycle bool) {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}

	inDegree := make(map[string]int, len(ids))
	dependents := make(map[string][]string, len(ids))
	for _, id := range ids {
		inDegree[id] = 0
	}
	for _, id := range ids {
		for _, dep := range dependsOn[id] {
			if !set[dep] {
				continue
			}
			dependents[dep] = append(dependents[dep], id)
			inDegree[id]++
		}
	}

	queue := make([]string, 0, len(ids))
	for _, id := range ids {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0, len(ids))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)
		for _, next := range dependents[id] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(ids) {
		return ids, true
	}
	return result, false
}
