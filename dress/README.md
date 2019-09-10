# Dress


## Getting started

0. Read gqlgen docs by using:
```sh
$ cd /home/zyx/mygo/src/github.com/99designs/gqlgen/docs
$ hugo server
```

1. Edit gqlgen.yml and schema.graphql, generate files by using:
```sh
$ go run github.com/99designs/gqlgen
```

2. Implement same resolvers and move the functions to implemented.go.

3. Update gqlgen.yml and schema.graphql, delete resolver.go, and go to step 1.

## Tips

0. gqlen.yaml中可以配置使用自定义的结构体,schema中定义的字段会从结构体的成员变量和成员函数中寻找，
   未找到则自动生成相应的resolver函数。如果想对已存在的字段重新定义获取方法，则需要在gqlen.yaml中指定,参见：
   github.com/99designs/gqlgen/example/starwars/models.FriendsConnection.
  
1. 使用gorm时,用db.Table()与db.Model()的结果是不一样的，table会忽略对delete_at的判断处理，还有很多callbacks不会执行。