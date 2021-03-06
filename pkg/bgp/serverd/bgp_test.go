package serverd

import (
	"github.com/go-logr/logr/testing"
	bgpapi "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/nettool/iptables"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	bgpServer *BgpServer
)

var _ = BeforeSuite(func() {
	By("Init gobgp server and config")
	bgpOptions := &BgpOptions{
		GrpcHosts: ":50052",
	}

	bgpServer = NewBgpServer(bgpOptions, testing.NullLogger{}, iptables.NewFakeIPTables())
	err := bgpServer.HandleBgpGlobalConfig(&bgpapi.BgpConfSpec{
		As:       65003,
		RouterId: "10.0.255.254",
		Port:     17900,
	}, false)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("stop gobgp server")
	Expect(bgpServer.StopServer()).ShouldNot(HaveOccurred())
})

var _ = Describe("BGP routes test", func() {
	Context("Reconcile Routes", func() {
		It("Should generate right number", func() {
			a := generateIdentifier("192.168.98.1")
			b := generateIdentifier("192.168.98.11")
			c := generateIdentifier("192.168.98.133")
			Expect(a).To(BeEquivalentTo(1))
			Expect(b).To(BeEquivalentTo(11))
			Expect(c).To(BeEquivalentTo(133))
		})
		It("Should correctly add/delete all routes", func() {
			ip := "100.100.100.100"
			nexthops := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

			By("Init gobgp should be empty")
			err, toAdd, toDelete := bgpServer.retriveRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(toAdd)).Should(Equal(3))
			Expect(len(toDelete)).Should(Equal(0))

			By("Add nexthops to gobgp")
			err = bgpServer.SetBalancer(ip, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			err, toAdd, toDelete = bgpServer.retriveRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(toAdd)).Should(Equal(0))
			Expect(len(toDelete)).Should(Equal(0))

			By("Append a nexthop to gobgp")
			nexthops = append(nexthops, "4.4.4.4")
			Expect(len(nexthops)).Should(Equal(4))
			err = bgpServer.SetBalancer(ip, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			err, toAdd, toDelete = bgpServer.retriveRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(toAdd)).Should(Equal(0))
			Expect(len(toDelete)).Should(Equal(0))

			By("Delete two nexthops from gobgp")
			nexthops = nexthops[:len(nexthops)-2]
			Expect(len(nexthops)).Should(Equal(2))
			err = bgpServer.SetBalancer(ip, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			err, toAdd, toDelete = bgpServer.retriveRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(toAdd)).Should(Equal(0))
			Expect(len(toDelete)).Should(Equal(0))

			By("Delete all nexthops from gobgp")
			Expect(bgpServer.DelBalancer(ip)).ShouldNot(HaveOccurred())
			err, toAdd, toDelete = bgpServer.retriveRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(toAdd)).Should(Equal(2))
			Expect(len(toDelete)).Should(Equal(0))
		})
	})

	Context("Create/Update/Delete BgpPeer", func() {
		It("Add BgpPeer", func() {
			Expect(bgpServer.AddOrUpdatePeer(&bgpapi.BgpPeerSpec{
				Config: bgpapi.NeighborConfig{
					PeerAs:          65001,
					NeighborAddress: "192.168.0.2",
				},
			})).ShouldNot(HaveOccurred())
		})

		It("Update BgpPeer", func() {
			Expect(bgpServer.AddOrUpdatePeer(&bgpapi.BgpPeerSpec{
				Config: bgpapi.NeighborConfig{
					PeerAs:          65002,
					NeighborAddress: "192.168.0.2",
				},
			})).ShouldNot(HaveOccurred())
		})

		It("Delete BgpPeer", func() {
			Expect(bgpServer.DeletePeer(&bgpapi.BgpPeerSpec{
				Config: bgpapi.NeighborConfig{
					NeighborAddress: "192.168.0.2",
				},
			})).ShouldNot(HaveOccurred())
		})
	})
})
