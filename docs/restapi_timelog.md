# Timelog
타임로그 관련 Rest API 사용법입니다.

<br>

#### Get

#### Post

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/checkmonthlystatus | 월별 결산 상태 확인 |  | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.10/api/checkmonthlystatus"` |
| /api/updatetimelog | 타임로그 업데이트 | status | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.10/api/updatetimelog?status=true"` |
| /api/resettimelog | 타임로그 리셋 | | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.10/api/resettimelog"` |


#### Delete
| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/rmtimelogbyid | ID가 작성한 타임로그 삭제 | id | `$ curl -H "Authorization: Basic <TOKEN>" -X DELETE "http://10.20.31.10/api/rmtimelogbyid?id=90"` |
| /api/rmtimelogbyproject | 프로젝트에 작성한 타임로그 삭제 | project | `$ curl -H "Authorization: Basic <TOKEN>" -X DELETE "http://10.20.31.10/api/rmtimelogbyproject?project=BEE%20BEC"` |