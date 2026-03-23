# Contributing

Issues and Pull Requests are welcome if they are well-researched and _not_ 
hallucinated by an LLM.

Obvious slop pull requests and issues will be closed without comment.

## Appreciated Methods of Contribution

1. Review pull requests that are open and provide feedback, not just approvals.
2. Send pull requests to implement accepted features or acknowledged bugs.

   a. Pull Requests for issues that have not been acknowledged will not be
      accepted until the issue is verified and acknowledged
   b. Duplicate pull requests for issues when there is one or more open/outstanding
      pull requests for an issue will be closed without comment, especially
      if they are obviously "agent" created.

3. Validating issues without the use of an LLM.

## Pull Request Standards

1. **Never** use conventional commit messages
2. Follow the example of the Linux Kernel Commit Message style guide, e.g.,
   [something akin to](https://gist.github.com/robertpainsi/b632364184e70900af4ab688decf6f53).
3. Run at least `make test` if not `make integration-test` locally
4. Run `make lint` locally
5. Run `pre-commit run --all` locally
6. (Optional) If adding a new dependency, run `make vulncheck` locally.
