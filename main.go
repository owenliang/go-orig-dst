package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"syscall"
)

const (
	SOL_IP                           = 0x0
	IP_ORIGDSTADDR                   = 0x14
)
func main() {
	var listener net.Listener
	var err error

	if listener, err = net.Listen("tcp4", "127.0.0.1:8080"); err == nil {
		var conn net.Conn
		for {
			// 接收连接
			if conn, err = listener.Accept(); err != nil {
				break
			}

			// 反射到TCP连接
			var tcpConn *net.TCPConn
			tcpConn, _ = conn.(*net.TCPConn)

			// 获取socket文件描述符
			var file *os.File
			file, _ = tcpConn.File()
			fd := file.Fd()

			var ip net.IP
			var port int
			// 系统调用getsockopt获取原始目标地址
			// 不要纳闷为啥是个IPV6调用，这是因为Golang在封装系统调用的时候不健全，用IPV6方法仅仅是因为它的IPV6结构体内存足够装下系统调用返回的数据而已
			var ipv6Mreq *syscall.IPv6Mreq
			if ipv6Mreq, err = syscall.GetsockoptIPv6Mreq(int(fd), SOL_IP, IP_ORIGDSTADDR); err == nil {
				// 实际ipv6Mreq这块内存被填充的是Linux C层面的结构体
				//struct sockaddr_in {
				//	sa_family_t    sin_family; /* address family: AF_INET */
				//	in_port_t      sin_port;   /* port in network byte order */
				//struct in_addr sin_addr;   /* internet address */
				//};
				//
				//	/* Internet address. */
				//struct in_addr {
				//	uint32_t       s_addr;     /* address in network byte order */
				//};

				// 所以4:8是sin_addr字段，也就是大端字节序的ipv4地址
				ip = net.IP(ipv6Mreq.Multiaddr[4:8])
				// 所以2:4是sin_port字段。也就是大端字节序的端口
				port = int(binary.BigEndian.Uint16(ipv6Mreq.Multiaddr[2:4]))
			} else {
				// 如果没取到, 则说明是原始目标IP就是本机
				addr := tcpConn.LocalAddr()
				tcpAddr, _ := net.ResolveTCPAddr(addr.Network(), addr.String())
				ip = tcpAddr.IP
				port = tcpAddr.Port
			}
			// 关闭dup复制的fd
			file.Close()

			fmt.Println("原始目标地址：%s:%d", ip.String(), port)
			// 关闭socket
			conn.Close()
		}
	}
}
