package trackers

func SumArray(arr []int) int64 {
	var sum int64
	for _, value := range arr {
		sum += int64(value)
	}
	return sum
}
