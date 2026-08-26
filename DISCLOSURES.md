# AI Usage Disclosure

**The Family Library project has been developed using local AI tools**. This includes usage of self-hosted models like Qwen 3.6 27B/35B A3B, Qwen 3.8 27B, Gemma 4 12B QAT, and Deepseek V4 Flash 0731 using self-hosted [Turnstone](https://github.com/turnstonelabs/turnstone) as my agent harness.

# Third-Party Integration Disclosures

This project may use dependencies with different development policies from third-parties. You can review the [Dependency Graph here](https://github.com/Toph4er/family-library/network/dependencies).

This project leverages the [Open Library API](https://openlibrary.org/developers/api) provided by the [Internet Archive](https://archive.org/) to retrieve information about the books in your collection. Personal details you provide about these books are never pushed TO Open Library, but Open Library could technically infer the books in your collection based on the queries made. You can always simply not use the ISBN scanning/lookup features to avoid this if it is a concern.

**No user data will ever be collected, sold, or shared with any third-parties by the maintainers of this project.**

**Please alert the maintainers of this project if an exception is found so it can be addressed or disclosed properly.**

# Data Privacy

Every effort has been taken by the maintainer, [@Toph4er](https://github.com/Toph4er), to ensure this project respects YOUR data privacy and sovereignty. This project includes zero telemetry or analytics tracking beyond what GitHub tracks by default for downloads, contributions, etc. You have the right to fully inspect the code base and verify these claims. If you do discover any violations of this assertion, please [file an Issue](https://github.com/Toph4er/family-library/issues) report with the details immediately.

When you spin this project up for yourself, it creates a new unique-to-you database instance that you alone control. If you decide to make contributions back to the project, please take care to practice good `.gitignore` hygiene to not leak your own data.

**No user data will ever be fed into an LLM by this project or its maintainers.**

# Contribution Disclosure Requirements

Contributors are welcome to use LLM tools for Pull Requests (PR). You are expected to use good discretion and personally validate functionality BEFORE submitting your PR. All PRs will be reviewed by the maintainer before merging into `main`.

**Contributors MUST disclose any AI/LLM usage in the PR including the models used, whether they are local or cloud hosted, and provide an attestation of Human review.**

# In Summary

I, [@Toph4er](https://github.com/Toph4er), built this project for my own family using local LLM models largely as an experiment that turned into something more - all outside the purview of prying Corporate hands. I hope this project can help your family read more books together and preserve precious memories.

**No software is perfect. This project is provided "as-is" with no warranty (implied or otherwise) of any kind.**

**This file is 100% human-written by me.**