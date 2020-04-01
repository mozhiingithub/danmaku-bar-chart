# danmaku-bar-chart
高能进度条是哔哩哔哩网页端的一个功能。本仓库基于“每秒弹幕数体现高能程度”的设定，用golang实现了一个自制的“高能进度条”，本质是每秒弹幕数的柱状图。柱状图通过[go-echarts](https://github.com/go-echarts/go-echarts)生成。
## 使用
```
go run danmaku-bar-chart.go https://www.bilibili.com/video/BVXXXXXXX
```
运行后，程序会生成一个以视频实际cid命名的html文件，打开后即可看到弹幕数柱状图，还可将光标移至某一柱，查看对应时间。
