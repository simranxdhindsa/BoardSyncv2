 The restart conversation button is now working correctly and properly triggers the assistant/start flow without failing.
 If the LLM sends long responses, proper text wrapping is now applied so content does not overflow or create horizontal scroll issues.
 After sending a message via voice input, the input field now clears correctly and no previous text remains in the textbox.
 Text mode layout is now properly centered on large screens and aligned as per design Figma design instead of appearing too wide.
 If avatar mode is disabled at the asset level, it will no longer appear in the AI display options, ensuring only valid modes are selectable.
 The avatar image mismatch issue has been fixed so the avatar shown in chat now correctly matches the avatar rendered for the asset level avatar.
 When switching from audio mode to text mode, TTS playback no longer continues and is properly stopped during the transition.
 The chat history now refreshes correctly after archiving or restarting a conversation, as the history API is triggered properly.
 Unnecessary history API calls are no longer triggered when replaying previously sent messages in course preview.
 The login page distortion issue on hovering theme and language icons has been resolved and layout remains stable.
 The course completion modal translation issue has been fixed and correct localized text is now shown.
 A proper “Generating Your Report” loading state is now shown instead of a blank screen while loading reports in Tune.
 The video and TTS playback behavior in Vivid mode has been corrected so both now work independently without pausing each other.
 The LiveChat End button no longer shows unnecessary hover effects, aligning with expected UI behavior.
 The active asset indicator in the section menu now has fully rounded visuals for both current and completed states.
 The learning preference is now synced across profile, course preview, and chat panel for consistent behavior.
 A chat panel close animation has been added in audio and avatar modes for smoother UI transitions.
 Avatar playback no longer stops when opening the section menu or clicking on AVATAR itself and continues smoothly.
 When opening the language change modal, any currently playing audio is now paused automatically because there the user have to Preview Audio for new Audio model selected for that particular Audio.
 Course card hover behavior has been stabilized on Dashboard and no longer causes layout jumps or distortion while scrolling.

Studio
 Asset-level teaser and description support has been added so users can now configure thumbnail and description directly within Studio.
 The Crafter API loop issue causing repeated triggers has been fixed and no repeated queries are executed.
 The Agentic mode 404 issue has been resolved and no unexpected redirects occur while accessing project metadata.
 Tooltip content across Studio has been corrected and now displays accurate and consistent information.
 Branding inconsistencies in Studio UI elements have been fixed and now follow correct theme configurations.
 A new subtype option has been added when the course type is set to Skills Lab.
 Undo and redo options have been introduced in the Ardoise Assistant to revert or restore refined AI responses.
 The 404 redirect issue when opening a project after changing chat language has been fixed.
 In edit mode, previously entered text content is now preserved correctly and no longer gets cleared unintentionally.
 The Ardoise logo in the rich text editor no longer disappears while generating content.
 A guidance indicator is now shown when course setup is incomplete, preventing confusion during “Submit for Review”.
 Support for uploading asset-level teaser/thumbnail via API has been added and is now handled correctly.
 Default project context is now automatically created and protected from deletion to prevent context-related issues.
 Special character support has been added for job roles, skills, and project names.
 Replacing a link-type resource now works correctly and shows properly as "Uploaded" status in both Studio and Mission Control.

Mission Control
 Bot settings payload (model, max tokens, temperature, top P) is now correctly sent during first time publishing of the bot.
 The issue where users got stuck in the Branding section while navigating to other Tabs like Employees, Overview, etc. has been resolved and navigation between tabs works correctly.
 Duplicate related/recommended courses are now prevented in Mission Control.
 Pagination APIs for team members and sub-teams have been implemented and integrated properly.
 The Hierarchy access logic has been implemented so managers now have access to both direct and indirect reports.
 The issue where removed team members remained visible until refresh has been fixed.
 A new Studio translations tab has been added under translations for better management.
 Search fields across UI, MC, and Studio now correctly reload data when cleared using backspace.
 Published Knowledge Base files now correctly show chunk entries in Mission Control.
 The Avatar preview API CSP issue has been resolved by allowing required policies.
 Usage analytics now correctly display TTS cost and output units, replacing invalid values with “0”.
 The Display Mode Change API is now wired and tracked in the organisation overview for Audio, Avatar, and Text modes.
 Bot versioning details now correctly display configuration settings like model, tokens, temperature, and top P.
 The deletion API (section, asset, assignment) no longer returns 500 errors and works as expected.
 Asset-level teaser and description support has also been added in Mission Control UI and APIs.

Backend / Platform
 The RAG Crafter loop issue has been fixed and repeated “Hi” query triggers no longer occur.
 The profile API access model has been improved by implementing a full reporting hierarchy for direct and indirect reports.
 Validation has been added to prevent duplicate kb_retriever entries in bot settings.
 The skill-score API now supports date filters, allowing users to select a custom date range.
 Duplicate team member addition is now prevented and related 500 errors have been resolved.

Hotfixes Planned (Monday)
The following fixes are already identified and will be deployed as hotfixes on Monday:
 Add course metadata descriptions (target audience, prerequisites, objectives) in Studio
 Fix Retake Course button redirection and add proper icon
 Add completion confirmation modal when navigating away from Tune
 Fix video controls hover behavior (quality & speed)
 Ensure course preview uses configured voice model per asset
 Add voice model configuration at asset level in Mission Control
 Fix glassmorphism UI issue when flat design is disabled in onboarding
 Resolve remaining linting issues across frontend[9:06 AM]Apologies for the delay in sending this update :zipper_mouth_face: