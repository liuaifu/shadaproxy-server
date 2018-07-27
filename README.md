# shadaproxy-server
一个用于穿透NAT，访问内网服务的工具

## 编译、运行
此服务需部署在公网。

    $ go get github.com/liuaifu/buffer
	$ git clone https://github.com/liuaifu/shadaproxy-server.git
	$ cd shadaproxy-server
	$ go build
	$ cp config.xml.sample config.xml
	$ vim config.xml
	$ ./shadaproxy-server -d
	$ ps aux|grep shadaproxy-server

## 配置示例
	<config>
		<port_for_agent>7789</port_for_agent>
		<!--目标服务列表，可以配多个服务-->
		<services>
			<service>
				<name>work_rdp</name>
				<!--与agent端配置的key相同-->
				<key>1234abcd</key>
				<!--提供给用户端访问的端口-->
				<port>13389</port>
			</service>
			<!--
			<service>
				<name>home_ftp</name>
				<key>5678</key>
				<port>10021</port>
			</service>
			-->
		</services>
	</config>

## Agent
Agent部署在目标服务所在机器或可以访问该服务的任何机器上。
代码地址：[https://github.com/liuaifu/shadaproxy-agent](https://github.com/liuaifu/shadaproxy-agent)
