package paginate

func ClampPagination(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}