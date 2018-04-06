## Changelog (Current version: 2.13.2)

-----------------

### 2.13.2 (2018 Apr 06)

* [2a4e2b0] Prepare for 2.13.2
* [7bc5c23] Copy the .ipa/.pkg to tmp folder (#44)

### 2.13.1 (2018 Mar 23)

* [2dbc557] prepare for 2.13.1
* [0a722c2] step definition update (#42)
* [0c98b64] Refactor ITMS configuration (#40)

### 2.13.0 (2018 Mar 01)

* [c60b316] prepare for 2.13.0
* [3e84702] Fix "ERROR: Could not start delivery: all transports failed diagnostics" (#38)

### 2.12.0 (2018 Jan 12)

* [d137dff] prepare for 2.12.0
* [b1f48e7] add AppPassword to make use of application-specific passwords (#34)
* [9e87c12] fix ci build (#36)

### 2.11.0 (2017 Oct 30)

* [9e970b0] Prepare for version 2.11.0
* [afd28fe] Change 'beta' -> 'review' (#32)

### 2.10.0 (2017 Oct 16)

* [1c9c410] prepare for 2.10.0
* [bb1d23b] new input: gemfile_path, gem version fixes, go dependencies with dep, (#29)
* [9c13a37] Add fastlane_version; lets you specify a specific version of fastlane to install and use (#28)

### 2.9.5 (2017 Sep 29)

* [af49e48] prepare for 2.9.5
* [668eb4b] add bundle_id, make app_id not required, add description text to specify that one of the two fields is required (#27)

### 2.9.4 (2017 Jul 05)

* [acdc839] prepare for 2.9.4
* [a9c668d] type tag updates

### 2.9.3 (2017 Jul 05)

* [5f7e43a] prepare for 2.9.3
* [cd55149] retry gem install and gem update commands (#25)

### 2.9.2 (2017 Apr 25)

* [80c4c43] Prepare for 2.9.2
* [a717c76] Added platform input (#23)

### 2.8.2 (2017 Jan 25)

* [3b5430e] TeamID&TeamName parameter check (#22)

### 2.8.1 (2017 Jan 16)

* [d87a434] prepare for 2.8.1
* [b3838ca] use rubycommand package (#21)
* [38b9eb5] step.yml typo fix

### 2.8.0 (2016 Dec 28)

* [06bd7ed] STEP_VERSION: 2.8.0
* [1ae2de3] Feature/deliver fastlane fix (#20)

### 2.7.3 (2016 Dec 19)

* [5acac8c] prepare for 2.7.3
* [2c2d70f] add macos tag (#19)

### 2.7.2 (2016 Dec 16)

* [57ab0b0] prep for v2.7.2
* [79da1a3] Feature/deliver update fixes (#18)

### 2.7.1 (2016 Dec 12)

* [085fd6d] prepare for 2.7.1
* [0276868] Implement missing team args (#17)

### 2.7.0 (2016 Dec 06)

* [5fb4350] prepare for 2.7.0
* [546f7e0] go-toolkit support (#16)

### 2.6.3 (2016 Nov 24)

* [abbd4ed] prepare for 2.6.3
* [eed73aa] print performed deliver command (#15)

### 2.6.2 (2016 Sep 21)

* [9eabedc] v2.6.2 prep - bitrise.yml cleanup
* [fbe67a5] Added TestFlight to more places in descriptions
* [e02e5ca] step title - include TestFlight

### 2.6.1 (2016 Aug 08)

* [482e43b] prep for v2.6.1
* [8b04fc6] Merge pull request #10 from ghuntley/patch-2
* [4743c25] corrected spelling mistake

### 2.6.0 (2016 May 25)

* [aaead97] prepare for release
* [00928b4] Merge pull request #8 from bitrise-io/team_id
* [7c984ab] team_id support

### 2.5.1 (2016 May 18)

* [2be7d7f] prepare for release
* [9662ece] step description updates

### 2.5.0 (2016 May 03)

* [3921296] prepare for release
* [5fd18c6] Merge pull request #7 from olegoid/master
* [204129c] Add Mac apps deployment support

### 2.4.1 (2016 Apr 13)

* [4b0b3c5] prepare for share
* [fe51a2e] Merge pull request #6 from bitrise-io/update_deliver
* [2806199] update log
* [5abcdc3] step.yml update
* [44ba436] update deliver gem

### 2.4.0 (2016 Mar 10)

* [0b27eb8] prepare for release
* [9c23149] Merge pull request #5 from Itelios/master
* [222d113] change version
* [20b0107] Add new parameter to have capacity to deliver with multiple team account
* [15944b2] STEP_GIT_VERION_TAG_TO_SHARE: 2.3.0

### 2.3.0 (2016 Jan 13)

* [270e924] bitrise.yml updated to be compatible with Bitrise CLI 1.3.0
* [904df42] Merge pull request #4 from blured2000/master
* [7995eca] tag fix
* [3599c6e] reverted custom urls
* [7b54505] updated tag
* [5269c4b] updated urls
* [3d8abbd] skipping metadata and screenshots is default
* [c73f98a] debugging fix
* [8a6bf75] update forked url for sharing
* [c1e54e8] default input values for test
* [8804fb6] skipping metadata and screenshot by default
* [ff7881f] updated step to use env vars for skipping metadata and screenshots
* [84148a9] added options for metadata and screenshot flags
* [40bf08b] ignore system file
* [198ae71] added share-this-step workflow

### 2.2.1 (2015 Dec 11)

* [16241dc] is_dont_change_value: false - removed; project_type_tags iOS

### 2.2.0 (2015 Oct 29)

* [2910b23] added support for OS X system ruby and brew installed ruby; is_expand: true for all inputs

### 2.1.0 (2015 Oct 17)

* [a5a44f2] debug for submit_for_beta input
* [727fc0e] added new flags
* [e267608] update for new deliver syntax
* [5a2a342] Merge pull request #2 from bazscsa/patch-1
* [0c063ee] Update step.yml

### 2.0.0 (2015 Sep 08)

* [8535358] bitrise stack related update
* [5380c58] Merge pull request #1 from gkiki90/update
* [5b94f0f] update

### 1.0.1 (2015 Apr 09)

* [8e00837] note about the undocumented & unsupported API usage, both in step.sh and in step.yml

-----------------

Updated: 2018 Apr 06