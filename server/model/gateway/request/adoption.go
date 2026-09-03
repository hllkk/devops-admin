package request

// AdoptionSearch 覆盖率/采用度查询(P3)。
// 与成本分析同筛选面(时间/部门子树/用户/Key/模型/供应商)，dimension/sort 字段在本域无意义
// (覆盖率无多维明细概念，部门/模型分布为独立端点)，故直接别名复用绑定结构，零重复定义。
type AdoptionSearch = CostAnalysisSearch
