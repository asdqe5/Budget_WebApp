# Project
프로젝트 관련 Rest API 사용법입니다.

<br>

#### Get

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/monthlyPurchaseCost | 프로젝트의 월별 구매 내역 가져오기 | id, date | `$ curl -H "Authorization: Basic <TOKEN>" -X GET "http://10.20.31.160/api/monthlyPurchaseCost?id=BEC&date=2020-11"` |

<br>

#### Post

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/setMonthlyPurchaseCost | 프로젝트의 월별 구매 내역 업데이트 | id, date, companyName{i}, detail{i}, expenses{i}, num | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.160/api/setMonthlyPurchaseCost?id=BEC&date=2020-06&companyName0=여기&detail0=저기&expenses0=1000&companyName1=저기&detail1=여기&expenses1=3000&num=2"` |

<br>

#### Delete

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/rmproject | 프로젝트 삭제 | id | `$ curl -H "Authorization: Basic <TOKEN>" -X DELETE "http://10.20.31.160/api/rmproject?id=BEE"` |