package driver

import "tei/internal/environment"

// DatabaseProviderBoundContractQueryExecutorは、特定のDatabase providerへ結び付いた
// ContractQueryExecutorがruntime compositionへ自身のproviderを明示するための任意interfaceである。
//
// API等のDatabase providerを持たないExecutorへ実装を強制しない。一方、SQL Executorのように
// provider固有である実装はこのinterfaceを実装し、Environment上のDatabase instance providerと
// 実行前に照合できるようにする。
type DatabaseProviderBoundContractQueryExecutor interface {
	ContractQueryExecutor
	DatabaseProvider() environment.DatabaseProvider
}
