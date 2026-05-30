import { memo, useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { Info, TrendingUp, TrendingDown, Minus } from 'lucide-react';
import {
  SkillLevel,
  RateCategory,
  SKILL_LEVEL_LABELS,
  formatCurrency,
  getCategoryTree,
  CategoryTreeNode
} from './types';

interface RateMatrixViewProps {
  rates: Record<string, RateCategory>;
  isLoading?: boolean;
}

interface FlatRate {
  skillLevel: string;
  category: string;
  subcategory?: string;
  amount: number;
  path: string;
}

export const RateMatrixView = memo(function RateMatrixView({ rates, isLoading = false }: RateMatrixViewProps) {
  // Memoize expensive calculations
  const flatRates = useMemo((): FlatRate[] => {
    const flatRatesArray: FlatRate[] = [];
    
    Object.entries(rates).forEach(([skillLevel, category]) => {
      const extractRates = (
        cat: RateCategory, 
        parentPath: string[] = [],
        mainCategory?: string
      ): void => {
        Object.entries(cat).forEach(([key, value]) => {
          const currentPath = [...parentPath, key];
          
          if (typeof value === 'number') {
            flatRatesArray.push({
              skillLevel,
              category: mainCategory || (parentPath[0] || key),
              subcategory: parentPath.length > 0 ? key : undefined,
              amount: value,
              path: currentPath.join('.')
            });
          } else {
            extractRates(value, currentPath, mainCategory || (parentPath.length === 0 ? key : parentPath[0]));
          }
        });
      };
      
      extractRates(category);
    });
    
    return flatRatesArray;
  }, [rates]);
  
  // Memoize grouped rates calculation
  const groupedRates = useMemo(() => {
    return flatRates.reduce((acc, rate) => {
      if (!acc[rate.category]) {
        acc[rate.category] = [];
      }
      acc[rate.category].push(rate);
      return acc;
    }, {} as Record<string, FlatRate[]>);
  }, [flatRates]);

  // Memoize skill levels
  const skillLevels = useMemo(() => Object.keys(rates), [rates]);
  
  // Memoize statistics calculation
  const statistics = useMemo(() => {
    if (flatRates.length === 0) return null;
    
    const amounts = flatRates.map(r => r.amount);
    const min = Math.min(...amounts);
    const max = Math.max(...amounts);
    const avg = Math.round(amounts.reduce((sum, amt) => sum + amt, 0) / amounts.length);
    
    const bySkillLevel = skillLevels.reduce((acc, level) => {
      const levelRates = flatRates.filter(r => r.skillLevel === level);
      if (levelRates.length > 0) {
        const levelAmounts = levelRates.map(r => r.amount);
        acc[level] = {
          min: Math.min(...levelAmounts),
          max: Math.max(...levelAmounts),
          avg: Math.round(levelAmounts.reduce((sum, amt) => sum + amt, 0) / levelAmounts.length),
          count: levelAmounts.length
        };
      }
      return acc;
    }, {} as Record<string, { min: number; max: number; avg: number; count: number }>);
    
    return { min, max, avg, total: flatRates.length, bySkillLevel };
  }, [flatRates, skillLevels]);

  const renderTreeView = () => {
    return (
      <div className="space-y-4">
        {Object.entries(rates).map(([skillLevel, category]) => {
          const tree = getCategoryTree(category);
          
          const renderNode = (node: CategoryTreeNode, level: number = 0): JSX.Element => {
            const indent = level * 24;
            
            if (node.isLeaf) {
              return (
                <div 
                  key={node.path.join('.')}
                  className="flex items-center justify-between p-2 border-l-2 border-muted"
                  style={{ marginLeft: indent }}
                >
                  <div className="flex-1">
                    <span className="typography-body-medium">{node.key}</span>
                    <div className="typography-body-small text-muted-foreground">
                      {node.path.join(' > ')}
                    </div>
                  </div>
                  <Badge variant="secondary" className="font-mono">
                    {formatCurrency(node.value as number)}
                  </Badge>
                </div>
              );
            }
            
            return (
              <div key={node.path.join('.')} className="space-y-2">
                <div 
                  className="flex items-center gap-2 p-2 bg-muted/50 rounded-xl"
                  style={{ marginLeft: indent }}
                >
                  <span className="font-medium">{node.key}</span>
                  <Badge variant="outline">
                    {(node.value as CategoryTreeNode[]).length} mục
                  </Badge>
                </div>
                <div className="space-y-1">
                  {(node.value as CategoryTreeNode[]).map(child => 
                    renderNode(child, level + 1)
                  )}
                </div>
              </div>
            );
          };
          
          if (tree.length === 0) {
            return (
              <Card key={skillLevel}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    {SKILL_LEVEL_LABELS[skillLevel as SkillLevel]}
                    <Badge variant="outline">Trống</Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-center py-4 text-muted-foreground">
                    Chưa có cấu hình nào
                  </div>
                </CardContent>
              </Card>
            );
          }
          
          return (
            <Card key={skillLevel}>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  {SKILL_LEVEL_LABELS[skillLevel as SkillLevel]}
                  <Badge variant="outline">
                    {tree.reduce((count, node) => {
                      const countNodes = (n: CategoryTreeNode): number => {
                        if (n.isLeaf) return 1;
                        return (n.value as CategoryTreeNode[]).reduce((sum, child) => sum + countNodes(child), 0);
                      };
                      return count + countNodes(node);
                    }, 0)} mức lương
                  </Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {tree.map(node => renderNode(node))}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    );
  };

  const renderTableView = () => {
    if (flatRates.length === 0) {
      return (
        <div className="text-center py-8 text-muted-foreground">
          <Info className="w-12 h-12 mx-auto mb-4 text-muted-foreground/50" />
          <p>Chưa có dữ liệu mức lương để hiển thị</p>
        </div>
      );
    }
    
    // Group by category for table display
    const categories = Object.keys(groupedRates);
    
    return (
      <div className="space-y-6">
        {categories.map(category => (
          <Card key={category}>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                {category}
                <Badge variant="outline">
                  {groupedRates[category].length} mức lương
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Trình độ</TableHead>
                    <TableHead>Danh mục con</TableHead>
                    <TableHead className="text-right">Mức lương</TableHead>
                    <TableHead>So sánh</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {groupedRates[category]
                    .sort((a, b) => a.amount - b.amount)
                    .map((rate, index) => {
                      const isLowest = index === 0;
                      const isHighest = index === groupedRates[category].length - 1;
                      
                      return (
                        <TableRow key={`${rate.skillLevel}-${rate.path}`}>
                          <TableCell>
                            <Badge variant="secondary">
                              {SKILL_LEVEL_LABELS[rate.skillLevel as SkillLevel]}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <span className="typography-body-medium">
                              {rate.subcategory || rate.path}
                            </span>
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {formatCurrency(rate.amount)}
                          </TableCell>
                          <TableCell>
                            {groupedRates[category].length > 1 && (
                              <div className="flex items-center gap-1">
                                {isLowest && (
                                  <Badge variant="outline" className="text-blue-600 border-blue-600">
                                    <TrendingDown className="w-3 h-3 mr-1" />
                                    Thấp nhất
                                  </Badge>
                                )}
                                {isHighest && (
                                  <Badge variant="outline" className="text-green-600 border-green-600">
                                    <TrendingUp className="w-3 h-3 mr-1" />
                                    Cao nhất
                                  </Badge>
                                )}
                                {!isLowest && !isHighest && (
                                  <Badge variant="ghost">
                                    <Minus className="w-3 h-3" />
                                  </Badge>
                                )}
                              </div>
                            )}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  };

  const renderStatisticsView = () => {
    if (!statistics) {
      return (
        <div className="text-center py-8 text-muted-foreground">
          <Info className="w-12 h-12 mx-auto mb-4 text-muted-foreground/50" />
          <p>Chưa có dữ liệu để tính toán thống kê</p>
        </div>
      );
    }
    
    return (
      <div className="space-y-6">
        {/* Overall Statistics */}
        <Card>
          <CardHeader>
            <CardTitle>Thống kê tổng quan</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center p-4 border rounded-xl">
                <div className="typography-headline-large text-green-600">
                  {formatCurrency(statistics.max)}
                </div>
                <div className="typography-body-medium text-muted-foreground">Cao nhất</div>
              </div>
              
              <div className="text-center p-4 border rounded-xl">
                <div className="typography-headline-large text-blue-600">
                  {formatCurrency(statistics.avg)}
                </div>
                <div className="typography-body-medium text-muted-foreground">Trung bình</div>
              </div>
              
              <div className="text-center p-4 border rounded-xl">
                <div className="typography-headline-large text-orange-600">
                  {formatCurrency(statistics.min)}
                </div>
                <div className="typography-body-medium text-muted-foreground">Thấp nhất</div>
              </div>
              
              <div className="text-center p-4 border rounded-xl">
                <div className="typography-headline-large text-purple-600">
                  {statistics.total}
                </div>
                <div className="typography-body-medium text-muted-foreground">Tổng mức</div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* By Skill Level */}
        <Card>
          <CardHeader>
            <CardTitle>Thống kê theo trình độ</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {Object.entries(statistics.bySkillLevel).map(([skillLevel, stats]) => (
                <div key={skillLevel} className="p-4 border rounded-xl">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-semibold">
                      {SKILL_LEVEL_LABELS[skillLevel as SkillLevel]}
                    </h3>
                    <Badge variant="outline">{stats.count} mức lương</Badge>
                  </div>
                  
                  <div className="grid grid-cols-3 gap-4 typography-body-medium">
                    <div>
                      <div className="text-muted-foreground">Cao nhất</div>
                      <div className="font-medium">{formatCurrency(stats.max)}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Trung bình</div>
                      <div className="font-medium">{formatCurrency(stats.avg)}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Thấp nhất</div>
                      <div className="font-medium">{formatCurrency(stats.min)}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  };

  // Loading skeleton
  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex space-x-1">
          <Skeleton className="h-10 flex-1" />
          <Skeleton className="h-10 flex-1" />
          <Skeleton className="h-10 flex-1" />
        </div>
        <div className="space-y-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-24" />
        </div>
      </div>
    );
  }

  if (Object.keys(rates).length === 0) {
    return (
      <Alert>
        <Info className="h-4 w-4" />
        <AlertDescription>
          Chưa có cấu hình mức lương nào. Hãy bắt đầu thêm mức lương đầu tiên.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-4">
      <Tabs defaultValue="tree" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="tree">Cây cấu trúc</TabsTrigger>
          <TabsTrigger value="table">Bảng so sánh</TabsTrigger>
          <TabsTrigger value="stats">Thống kê</TabsTrigger>
        </TabsList>

        <TabsContent value="tree" className="mt-4">
          {renderTreeView()}
        </TabsContent>

        <TabsContent value="table" className="mt-4">
          {renderTableView()}
        </TabsContent>

        <TabsContent value="stats" className="mt-4">
          {renderStatisticsView()}
        </TabsContent>
      </Tabs>
    </div>
  );
});