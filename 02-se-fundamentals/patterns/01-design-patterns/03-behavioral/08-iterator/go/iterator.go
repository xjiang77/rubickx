package iterator

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type PageIterator struct {
	pages                         []any
	pageIndex, itemIndex, fetched int
	items                         []any
}

func (i *PageIterator) Next() (int, bool, error) {
	for i.itemIndex >= len(i.items) {
		if i.pageIndex >= len(i.pages) {
			return 0, false, nil
		}
		page := i.pages[i.pageIndex]
		i.pageIndex++
		if failure, ok := page.(map[string]any); ok {
			return 0, false, &PatternError{code: failure["error"].(string)}
		}
		i.items = page.([]any)
		i.itemIndex = 0
		i.fetched++
	}
	value := int(i.items[i.itemIndex].(float64))
	i.itemIndex++
	return value, true, nil
}
func Evaluate(input map[string]any) (any, error) {
	pages, _ := input["pages"].([]any)
	iterator := &PageIterator{pages: pages}
	items := []int{}
	exhausted := false
	take := int(input["take"].(float64))
	for n := 0; n < take; n++ {
		value, ok, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			exhausted = true
			break
		}
		items = append(items, value)
	}
	return map[string]any{"items": items, "fetched_pages": iterator.fetched, "exhausted": exhausted}, nil
}
