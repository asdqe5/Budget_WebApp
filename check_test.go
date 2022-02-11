// 프로젝트 결산 프로그램
//
// Description : 레귤러 익스프레션 테스트 스크립트

package main

import (
	"testing"
)

// 연봉 형식을 테스트하기 위한 함수
func Test_checkSalary(t *testing.T) {
	cases := []struct {
		salary string
		want   bool
	}{{
		salary: "2020:2400",
		want:   true,
	}, {
		salary: "2020:2400,2020:2400",
		want:   true,
	}, {
		salary: "2020:2400, 2020:2400", // 띄어쓰기가 포함된 경우
		want:   false,
	}, {
		salary: ":2400", // 날짜를 쓰지 않은 경우
		want:   false,
	}, {
		salary: "2020:", // 연봉을 쓰지 않은 경우
		want:   false,
	}, {
		salary: "2020:2400,", // , 뒤에 값이 없는 경우
		want:   false,
	}, {
		salary: "2020:2400,2020", // 연봉을 쓰지 않은 경우
		want:   false,
	}, {
		salary: "", // 빈문자열인 경우
		want:   false,
	},
	}

	for _, c := range cases {
		b := regexSalary.MatchString(c.salary)
		if c.want != b {
			t.Fatalf("Test_checkSalary(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.salary, c.want, b)
		}
	}
}

// map 형식을 테스트하기 위한 함수
func Test_checkMap(t *testing.T) {
	cases := []struct {
		str  string
		want bool
	}{{
		str:  "key:value",
		want: true,
	}, {
		str:  "2020-06:2400,2020-07:2400",
		want: true,
	}, {
		str:  "key:value, key:value", // 띄어쓰기가 포함된 경우
		want: false,
	}, {
		str:  "key:value,key:", // : 뒤에 쓰지 않은 경우
		want: false,
	}, {
		str:  ":value", // key를 쓰지 않은 경우
		want: false,
	}, {
		str:  "key:", // value를 쓰지 않은 경우
		want: false,
	}, {
		str:  "key:value,", // , 뒤에 값이 없는 경우
		want: false,
	}, {
		str:  "key:value,key", // value를 쓰지 않은 경우
		want: false,
	}, {
		str:  "", // 빈문자열인 경우
		want: false,
	},
	}

	for _, c := range cases {
		b := regexMap.MatchString(c.str)
		if c.want != b {
			t.Fatalf("Test_checkMap(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.str, c.want, b)
		}
	}
}

// date 형식을 테스트하기 위한 함수
func Test_checkDate(t *testing.T) {
	cases := []struct {
		date string
		want bool
	}{{
		date: "2020-07",
		want: true,
	}, {
		date: "2020-13", // 13월
		want: false,
	}, {
		date: "2020-", // 월을 입력하지 않은 경우
		want: false,
	}, {
		date: "-07", // 연도를 입력하지 않은 경우
		want: false,
	},
	}

	for _, c := range cases {
		b := regexDate.MatchString(c.date)
		if c.want != b {
			t.Fatalf("Test_checkDate(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.date, c.want, b)
		}
	}
}

// dept 형식을 테스트하기 위한 함수
func Test_checkDept(t *testing.T) {
	cases := []struct {
		dept string
		want bool
	}{{
		dept: "Comp1",
		want: true,
	}, {
		dept: "3D",
		want: true,
	}, {
		dept: "3D+FX",
		want: true,
	}, {
		dept: "RD_Manager", // _가 포함된 경우
		want: false,
	}, {
		dept: "comp!", // !가 포함된 경우
		want: false,
	},
	}

	for _, c := range cases {
		b := regexDept.MatchString(c.dept)
		if c.want != b {
			t.Fatalf("Test_checkDept(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.dept, c.want, b)
		}
	}
}

// 영문, 숫자만 있는 형식을 테스트하기 위한 함수
func Test_checkWord(t *testing.T) {
	cases := []struct {
		word string
		want bool
	}{{
		word: "Comp1",
		want: true,
	}, {
		word: "3D",
		want: true,
	}, {
		word: "3D+FX", // +가 포함된 경우
		want: false,
	}, {
		word: "RD_Manager", // _가 포함된 경우
		want: false,
	}, {
		word: "comp!", // !가 포함된 경우
		want: false,
	},
	}

	for _, c := range cases {
		b := regexWord.MatchString(c.word)
		if c.want != b {
			t.Fatalf("Test_checkWord(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.word, c.want, b)
		}
	}
}

// 프로젝트 이름 형식을 테스트하기 위한 함수
func Test_checkProject(t *testing.T) {
	cases := []struct {
		project string
		want    bool
	}{{
		project: "BEE",
		want:    true,
	}, {
		project: "RND2020",
		want:    true,
	}, {
		project: "CM_ART",
		want:    true,
	}, {
		project: "BEE BEC", // 띄어쓰기가 포함된 경우
		want:    false,
	}, {
		project: "bee", // 소문자가 포함된 경우
		want:    false,
	}, {
		project: "BEE!", // !가 포함된 경우
		want:    false,
	}, {
		project: "강철비", // 한글이 포함된 경우
		want:    false,
	},
	}

	for _, c := range cases {
		b := regexProject.MatchString(c.project)
		if c.want != b {
			t.Fatalf("Test_checkSGPermission(): 입력 값: %v, 원하는 값: %v, 얻은 값: %v\n", c.project, c.want, b)
		}
	}
}
