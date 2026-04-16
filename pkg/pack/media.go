package pack

const (
	MediaTypeModulePack = "application/vnd.kbi.modulepack.v1.tar"
	MediaTypeBPFPack    = "application/vnd.kbi.bpfpack.v1.tar"

	AnnotationPackType     = "io.multikernel.kbi.pack.type"
	AnnotationPackForKBIID = "io.multikernel.kbi.pack.for_kbi_id"
	AnnotationPackForKver  = "io.multikernel.kbi.pack.for_kver"
	AnnotationPackContents = "io.multikernel.kbi.pack.contents"
	AnnotationPackRequires = "io.multikernel.kbi.pack.requires"
)

type PackType string

const (
	PackTypeModule PackType = "modulepack"
	PackTypeBPF    PackType = "bpfpack"
)
