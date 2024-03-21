package utils

import (
	"fmt"
	"math"
)

// 计算平均值
func CalculateMeanInt64(data []int64) float64 {
	var total int64
	for _, value := range data {
		total += value
	}
	return float64(total) / float64(len(data))
}

// 计算平均值
func CalculateMeanFloat64(data []float64) float64 {
	total := 0.0
	for _, value := range data {
		total += value
	}
	return total / float64(len(data))
}

// 计算标准差
func CalculateStandardDeviationInt64(data []int64) float64 {
	if len(data) == 0 {
		return 0
	}
	mean := CalculateMeanInt64(data)
	sumOfSquaredDifferences := 0.0
	for _, value := range data {
		difference := float64(value) - mean
		sumOfSquaredDifferences += difference * difference
	}
	return math.Sqrt(sumOfSquaredDifferences / float64(len(data)))
}

// 所有视频帧和音频帧的帧号列出
func GetFrameNumberString(videoFrameNo []int64, audioFrameNo []int64) []string {
	var tmparr1, tmparr2 []int64
	var res []string
	i, j := 0, 0
	for i < len(videoFrameNo) && j < len(audioFrameNo) {
		if videoFrameNo[i] < audioFrameNo[j] {
			if len(tmparr1) > 0 && videoFrameNo[i]-1 != tmparr1[len(tmparr1)-1] {
				if len(tmparr1) == 1 {
					res = append(res, fmt.Sprintf("视频帧:%d   ", tmparr1[0]))
				} else {
					res = append(res, fmt.Sprintf("视频帧:%d-%d   ", tmparr1[0], tmparr1[len(tmparr1)-1]))
				}
				tmparr1 = tmparr1[:0]
			}
			tmparr1 = append(tmparr1, videoFrameNo[i])
			i++
		} else {
			if len(tmparr2) > 0 && audioFrameNo[j]-1 != tmparr2[len(tmparr2)-1] {
				if len(tmparr2) == 1 {
					res = append(res, fmt.Sprintf("音频帧:%d   ", tmparr2[0]))
				} else {
					res = append(res, fmt.Sprintf("音频帧:%d-%d   ", tmparr2[0], tmparr2[len(tmparr2)-1]))
				}
				tmparr2 = tmparr2[:0]
			}
			tmparr2 = append(tmparr2, audioFrameNo[j])
			j++
		}
	}

	if len(tmparr1) > 0 && len(tmparr2) > 0 {
		if tmparr1[0] < tmparr2[0] {
			res = append(res, fmt.Sprintf("视频帧:%d-%d", tmparr1[0], tmparr1[len(tmparr1)-1]))
			res = append(res, fmt.Sprintf("音频帧:%d-%d", tmparr2[0], tmparr2[len(tmparr2)-1]))
		} else {
			res = append(res, fmt.Sprintf("音频帧:%d-%d", tmparr2[0], tmparr2[len(tmparr2)-1]))
			res = append(res, fmt.Sprintf("视频帧:%d-%d", tmparr1[0], tmparr1[len(tmparr1)-1]))
		}
	} else if len(tmparr1) > 0 {
		res = append(res, fmt.Sprintf("视频帧:%d-%d", tmparr1[0], tmparr1[len(tmparr1)-1]))
	} else if len(tmparr2) > 0 {
		res = append(res, fmt.Sprintf("音频帧:%d-%d", tmparr2[0], tmparr2[len(tmparr2)-1]))
	}

	if i < len(videoFrameNo) {
		res = append(res, fmt.Sprintf("视频帧:%d-%d", videoFrameNo[i], videoFrameNo[len(videoFrameNo)-1]))
	}
	if j < len(audioFrameNo) {
		res = append(res, fmt.Sprintf("音频帧:%d-%d   ", audioFrameNo[j], audioFrameNo[len(audioFrameNo)-1]))
	}
	return res
}
