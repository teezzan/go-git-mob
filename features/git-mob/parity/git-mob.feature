Feature: git-mob.spec

  Background:
    Given I have installed go-git-mob into "local_bin" within the current directory
    And I look for executables in "local_bin" within the current directory
    And I successfully run `git config --global user.name "Jane Doe"`
    And I successfully run `git config --global user.email "jane@example.com"`

  Scenario: -h prints help
    When I successfully run `git mob -h`
    Then the output should contain "Usage"
    And the output should contain "Flags"
    And the output should contain "Examples"

  #@announce-stderr
  @not-windows
  Scenario: --help prints help
    # --help is intercepted by the git plugin launcher which returns a 404 (help not found)
    When I run `git mob --help`
    Then the output should contain "No manual entry for git-mob"

  Scenario: -v prints version
    When I successfully run `git mob -v`
    Then the output should match /\d.\d.\d/

  Scenario: --version prints version
    When I successfully run `git mob --version`
    Then the output should match /\d.\d.\d/

  Scenario: --list prints a list of avaialable co-authors
    Given a file named "~/.git-coauthors" with:
      """
      {
        "coauthors": {
          "ad": {
            "name": "Amy Doe",
            "email": "amy@example.com"
          },
          "bd": {
            "name": "Bob Doe",
            "email": "bob@example.com"
          }
        }
      }
      """
    When I successfully run `git mob --list`
    Then the output should contain:
      """
      ad Amy Doe amy@example.com
      bd Bob Doe bob@example.com
      """

# @wip
# @announce-stdout
