<!-- jitctx audit | profile: spring-boot-hexagonal -->

## Sintatic Violations

## Module: user

- 🔴 ERROR [rest-controller-path-mismatch] src/main/java/com/app/user/UserController.java — Class @RestController must live under adapter/in/web/
  suggestion: Move src/main/java/com/app/user/UserController.java to a path containing "/adapter/in/web/"
- 🔴 ERROR [entity-path-mismatch] src/main/java/com/app/user/UserDB.java — Class @Entity must live under domain/
  suggestion: Move src/main/java/com/app/user/UserDB.java to a path containing "/domain/"
- 🔴 ERROR [domain-leak] src/main/java/com/app/user/domain/User.java — Files under domain/ must not import org.springframework.*
  suggestion: Remove the Spring import from src/main/java/com/app/user/domain/User.java; move framework code to an adapter
- 🟡 WARNING [port-naming] src/main/java/com/app/user/port/in/CreateUser.java — Interface in port/in/ must end with UseCase
  suggestion: Rename CreateUser to end with UseCase
- 🔴 ERROR [adapter-injection] src/main/java/com/app/user/service/UserService.java — Service must inject the output port, not an adapter implementation directly
  suggestion: In src/main/java/com/app/user/service/UserService.java, replace the adapter field type with the corresponding output port interface

## Semantic Analysis

Semantic analysis not enabled. Future versions of jitctx may support deeper checks via analyzer plugins.

