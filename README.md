## Условие

Система для добавления и чтения постов и комментариев с использованием GraphQ, аналогичная комментариям к постам на популярных платформах, таких как Хабр или Reddit.

Характеристики системы постов:

•	Можно просмотреть список постов.

•	Можно просмотреть пост и комментарии под ним.

•	Пользователь, написавший пост, может запретить оставление комментариев к своему посту.

Характеристики системы комментариев к постам:

•	Комментарии организованы иерархически, позволяя вложенность без ограничений.

•	Длина текста комментария ограничена до, например, 2000 символов.

•	Система пагинации для получения списка комментариев.

(*) Дополнительные требования для реализации через GraphQL Subscriptions:

•	Комментарии к постам должны доставляться асинхронно, т.е. клиенты, подписанные на определенный пост, должны получать уведомления о новых комментариях без необходимости повторного запроса.

Требования к реализации:
•	Система должна быть написана на языке Go.

•	Использование Docker для распространения сервиса в виде Docker-образа.

•	Должно быть реализовано 2 варианта хранение данных: в памяти (in-memory) и в PostgreSQL.

•	Покрытие реализованного функционала unit-тестами.

## Запуск контейнера

```bash
STORAGE_TYPE=postgres docker compose -f build/docker-compose.yml up -d --build
```

По умолчанию STORAGE_TYPE=in-memory.

## Тестирование

1. Создание поста

```bash
mutation {
  addPost(title: "My First Post", content: "Hello, GraphQL!", allowComments: true) {
    id
    title
    content
    allowComments
  }
}
```

2. Получение списка постов

```bash
query {
  posts {
    id
    title
    content
    allowComments
  }
}
```

3. Получение поста по ID

```bash
query {
  post(id: "12345") {
    id
    title
    content
    allowComments
  }
}
```

4. Добавление комментария к посту

```bash
mutation {
  addComment(postId: "12345", parentId: null, content: "This is my first comment!") {
    id
    postId
    parentId
    content
    createdAt
  }
}
```

5. Получение комментариев с пагинацией

```bash
query {
  comments(postId: "12345", limit: 5, offset: 0) {
    id
    postId
    parentId
    content
    createdAt
  }
}
```

6. Подписка на комментарии к посту

```bash
subscription {
  commentAdded(postId: "12345") {
    id
    postId
    parentId
    content
    createdAt
  }
}
```