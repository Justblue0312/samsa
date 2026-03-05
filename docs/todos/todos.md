Here is the list of todos for Samsa SAAS.

# Backend

- [x] Rework on:
    - [x] /internal/infra/session
    - [x] /pkgs/ratelimit
- [x] Complete the ws server
- [ ] Complete the APIs for, since those models related and can be based for the following components:
    - [x] Submission.
    - [x] Comment.
    - [x] Tag.
    - [x] Author.
    - [x] Notification.
- [x] Move mocks into /feature/<entity>/mocks for better management.
- [ ] Rewrite the testkit to setup test containers and mock dependencies more easily.
- [x] Split the apierror and respond in two packages.
- [ ] Re-evaluate the submission APIs and its related APIs.
- [ ] Improve the error handling in existing packages.
- [ ] Complete APIs for the main modules.
    - [ ] Story
    - [ ] Genre
    - [ ] Document
    - [ ] Chapter
    - [ ] Flag
    - [ ] Spinnet

# Frontend

- [ ] Implement the frontend with Solid.js/SolidStart/ReactJS + Tanstack Query

# Kit

- [ ] Implement the proto services to visualize which services we need to implement.
- [ ] Start to formming the based project with Fastapi + SQLAlchemy + ConnectRPC + Pydantic/Pydantic-AI and more.
