# Software Engineering Radio legacy code snapshot

Source URL: https://www.se-radio.net/2020/09/episode-429-michael-feathers-on-working-effectively-with-legacy-code/

This snapshot is prepared for harness evaluation. It is a compressed description of the source, not a verbatim copy.

- The discussion frames legacy code not as "old code" but as code that is hard to change safely. The practical problem is lack of fast feedback.
- A recurring theme is building a safety net before making aggressive changes. That usually means tests, but it can also mean seams, smaller change sets, or instrumentation that shortens the feedback loop.
- The speaker emphasizes seams: places where behavior can be changed without editing every dependency at once. Seams make small refactor steps possible in codebases that feel frozen.
- Another theme is reducing the emotional size of a change. Instead of one large rewrite, make a small refactor, check feedback, and repeat.
- This material is action-oriented and conversational. It is likely better as a Chinese blog article with concrete rules of thumb than as a full interactive course.
- The best downstream artifact would summarize the main heuristics, show how to start with tests or seams, and explain how to create a reliable feedback loop in messy systems.
