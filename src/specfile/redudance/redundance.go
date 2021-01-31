package redundance

import (
	"fmt"
)

// 检查组中所有的数据分片是不是符合要求,不然无法生成冗余分片
func checkFragmentGroup(fragmentGroup [][]byte) error {
	var err error = nil
	// 只有组内有多于等于两个数据分片才能生成冗余分片
	if len(fragmentGroup) < 2 {
		err = fmt.Errorf("组内少于两个数据分片,无法生成冗余组")
		fmt.Println(err)
		return err
	}
	// 检查所有数据分片是不是长度相同,不相同则无法生成冗余分片
	basicLength := len(fragmentGroup[0])
	for i := 1; i < len(fragmentGroup); i++ {
		if len(fragmentGroup[i]) != basicLength {
			err = fmt.Errorf("组内数据分片长度不相同,无法生成冗余组")
			fmt.Println(err)
			return err
		}
	}
	return err
}

// 利用一个组中所有的数据分片里面的数据,生成一个冗余分片.所有的数据分片必须长度相同
func GenerateRedundanceFragment(fragmentGroup *[][]byte) error {
	var err error
	if checkFragmentGroup(*fragmentGroup) != nil {
		err = fmt.Errorf("组内的数据分片不规范")
		fmt.Println(err)
		return err
	}
	var redundanceFragment []byte
	fragmentNum := len(*fragmentGroup)         // 数据分片的数目
	fragmentLength := len((*fragmentGroup)[0]) // 每个数据分片的长度
	for i := 0; i < fragmentLength; i++ {
		var redundanceBit byte = 0
		for j := 0; j < fragmentNum; j++ {
			bit := (*fragmentGroup)[j][i]
			redundanceBit = redundanceBit ^ bit
		}
		redundanceFragment = append(redundanceFragment, redundanceBit)
	}
	*fragmentGroup = append(*fragmentGroup, redundanceFragment)
	return err
}

// 利用收到的其他数据分片和冗余分片,还原出丢失的数据分片
func RestoreLostFragment(damagedDataFragmentGroup [][]byte, redundanceFragment []byte) []byte {
	var recoveryGroup [][]byte
	var recoveryFragment []byte
	recoveryGroup = append(damagedDataFragmentGroup, redundanceFragment)
	fragmentNum := len(recoveryGroup)       // 数据分片的数目
	fragmentLength := len(recoveryGroup[0]) // 每个数据分片的长度
	for i := 0; i < fragmentLength; i++ {
		var recoveryBit byte = 0
		for j := 0; j < fragmentNum; j++ {
			bit := recoveryGroup[j][i]
			recoveryBit = recoveryBit ^ bit
		}
		recoveryFragment = append(recoveryFragment, recoveryBit)
	}
	return recoveryFragment
}
