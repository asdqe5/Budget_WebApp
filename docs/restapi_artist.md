# Artist
아티스트 관련 Rest API 사용법입니다.

<br>

#### Get

#### Post

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/addartistvfx | VFX팀 아티스트 추가 | id, salary | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.160/api/addartistvfx?id=90&salary=2019:2400,2020:2400"` |
| /api/addartistcm | CM팀 아티스트 추가 | id, team, name, salary | `$ curl -H "Authorization: Basic <TOKEN>" -X POST "http://10.20.31.160/api/addartistcm?id=90&team=3D&name=로드&salary=2019:2400,2020:2400"` |

#### Delete

| URI | Description | Attributes | Curl Example |
| :--: | :--: | :--: | :--: |
| /api/rmartist | 아티스트 삭제 | id | `$ curl -H "Authorization: Basic <TOKEN>" -X DELETE "http://10.20.31.160/api/rmartist?id=90"` |