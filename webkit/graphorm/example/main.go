package main

import (
	"github.com/99designs/gqlgen/handler"
	"github.com/friendsofgo/graphiql"
	"github.com/gin-gonic/gin"
	"github.com/zltgo/webkit/graphorm/example/graph"
)

func main() {
	cfg := graph.Config{Resolvers: &graph.Resolver{}}

	//限制查询复杂度，将数组展开计算
	//countComplexity := func(childComplexity, count int) int {
	//return count * childComplexity
	//}
	//cfg.Complexity.Query.Posts = countComplexity
	//cfg.Complexity.Post.Related = countComplexity

	graphqlHandler := handler.GraphQL(
		graph.NewExecutableSchema(cfg),
		handler.ComplexityLimit(100),        //限制递归查询次数为100次，数组计算长度
		handler.IntrospectionEnabled(false), //关闭内省
		handler.CacheSize(1000),             //自带一个缓存,只是缓存解析查询语句的中间结果
	)
	graphiqlHandler, err := graphiql.NewGraphiqlHandler("/graphql")
	if err != nil {
		panic(err)
	}
	playgroundHandler := handler.Playground("GraphQL playground", "/graphql")

	// Setting up Gin
	r := gin.Default()
	r.POST("/graphql", gin.WrapH(graphqlHandler))
	r.GET("/graphiql", gin.WrapH(graphiqlHandler)) //the same as playground
	r.GET("/playground", gin.WrapH(playgroundHandler))

	r.Run()
}
