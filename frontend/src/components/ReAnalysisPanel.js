// import React, { useState } from 'react';
// import { RefreshCw, BarChart3, ArrowLeft, Zap, CheckCircle, AlertTriangle } from 'lucide-react';

// const ReAnalysisPanel = ({ 
//   selectedColumn, 
//   lastActionType, 
//   lastActionCount, 
//   onReAnalyze, 
//   onBackToDashboard,
//   loading,
//   error = null
// }) => {
//   const [selectedReAnalysisColumn, setSelectedReAnalysisColumn] = useState(selectedColumn);
//   const [localLoading, setLocalLoading] = useState(false);
//   const [localError, setLocalError] = useState(null);

//   const columns = [
//     { value: 'backlog', label: 'Backlog' },
//     { value: 'in_progress', label: 'In Progress' },
//     { value: 'dev', label: 'DEV' },
//     { value: 'stage', label: 'STAGE' },
//     { value: 'blocked', label: 'Blocked' },
//     { value: 'all_syncable', label: 'All Columns' }
//   ];

//   const getActionMessage = () => {
//     if (error || localError) {
//       return `❌ ${lastActionType || 'Action'} failed: ${error || localError}`;
//     }
    
//     const count = lastActionCount || 0;
//     switch (lastActionType) {
//       case 'sync':
//         return count > 0 
//           ? `🔄 ${count} tickets synced successfully!`
//           : `🔄 Sync completed - no tickets needed syncing!`;
//       case 'create':
//         return count > 0
//           ? `✅ ${count} tickets created successfully!`
//           : `✅ Create completed - no tickets needed creating!`;
//       case 'bulk_create':
//         return count > 0
//           ? `🚀 ${count} tickets created in bulk!`
//           : `🚀 Bulk create completed - no tickets needed creating!`;
//       case 'analysis':
//         return `📊 Analysis completed successfully!`;
//       default:
//         return lastActionCount !== undefined 
//           ? `✅ Action completed successfully! (${count} items processed)`
//           : '✅ Action completed successfully!';
//     }
//   };

//   const getActionIcon = () => {
//     if (error || localError) {
//       return <AlertTriangle className="w-6 h-6 text-red-600" />;
//     }
//     return <CheckCircle className="w-6 h-6 text-green-600" />;
//   };

//   const handleQuickReAnalyze = async () => {
//     setLocalLoading(true);
//     setLocalError(null);
    
//     try {
//       console.log('🔄 Quick re-analyze for column:', selectedColumn);
//       await onReAnalyze(selectedColumn);
//     } catch (err) {
//       console.error('❌ Quick re-analyze failed:', err);
//       setLocalError(err.message);
//     } finally {
//       setLocalLoading(false);
//     }
//   };

//   const handleCustomReAnalyze = async () => {
//     setLocalLoading(true);
//     setLocalError(null);
    
//     try {
//       console.log('🔄 Custom re-analyze for column:', selectedReAnalysisColumn);
//       await onReAnalyze(selectedReAnalysisColumn);
//     } catch (err) {
//       console.error('❌ Custom re-analyze failed:', err);
//       setLocalError(err.message);
//     } finally {
//       setLocalLoading(false);
//     }
//   };

//   const handleAnalyzeAll = async () => {
//     setLocalLoading(true);
//     setLocalError(null);
    
//     try {
//       console.log('🔄 Analyze all columns');
//       await onReAnalyze('all_syncable');
//     } catch (err) {
//       console.error('❌ Analyze all failed:', err);
//       setLocalError(err.message);
//     } finally {
//       setLocalLoading(false);
//     }
//   };

//   const isLoading = loading || localLoading;
//   const hasError = error || localError;

//   return (
//     <div className="glass-panel bg-white border border-gray-200 rounded-lg p-6 mb-6">
//       {/* Success/Error Message */}
//       <div className="flex items-center mb-6">
//         <div className="flex items-center mr-4">
//           {getActionIcon()}
//         </div>
//         <div className="flex-1">
//           <div className={`text-lg font-semibold mb-2 ${hasError ? 'text-red-900' : 'text-gray-900'}`}>
//             {getActionMessage()}
//           </div>
//           <p className="text-sm text-gray-600">
//             {hasError 
//               ? 'Please check your configuration and try again.'
//               : 'Choose how to proceed with your analysis workflow.'
//             }
//           </p>
//         </div>
//       </div>

//       {/* Error Details */}
//       {hasError && (
//         <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
//           <div className="flex items-center">
//             <AlertTriangle className="w-5 h-5 text-red-600 mr-2" />
//             <div>
//               <p className="text-red-800 font-medium">Action Failed</p>
//               <p className="text-red-700 text-sm mt-1">{error || localError}</p>
//             </div>
//           </div>
          
//           <div className="mt-3">
//             <button
//               onClick={() => {
//                 setLocalError(null);
//                 if (onReAnalyze) {
//                   handleQuickReAnalyze();
//                 }
//               }}
//               disabled={isLoading}
//               className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 flex items-center text-sm"
//             >
//               {isLoading ? (
//                 <>
//                   <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
//                   Retrying...
//                 </>
//               ) : (
//                 <>
//                   <RefreshCw className="w-4 h-4 mr-2" />
//                   Retry Analysis
//                 </>
//               )}
//             </button>
//           </div>
//         </div>
//       )}

//       {/* Quick Actions */}
//       <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
//         {/* Quick Re-analyze Same Column */}
//         <button
//           onClick={handleQuickReAnalyze}
//           disabled={isLoading}
//           className="glass-panel interactive-element p-4 rounded-lg border border-blue-200 bg-blue-50 hover:bg-blue-100 disabled:opacity-50 transition-all"
//         >
//           <div className="flex items-center justify-center mb-2">
//             <RefreshCw className={`w-5 h-5 text-blue-600 ${isLoading ? 'animate-spin' : ''}`} />
//           </div>
//           <div className="text-sm font-medium text-blue-900">
//             Re-analyze "{selectedColumn?.replace('_', ' ')?.toUpperCase() || 'SELECTED'}"
//           </div>
//           <div className="text-xs text-blue-700 mt-1">
//             Same column, fresh data
//           </div>
//         </button>

//         {/* Analyze All Columns */}
//         <button
//           onClick={handleAnalyzeAll}
//           disabled={isLoading}
//           className="glass-panel interactive-element p-4 rounded-lg border border-green-200 bg-green-50 hover:bg-green-100 disabled:opacity-50 transition-all"
//         >
//           <div className="flex items-center justify-center mb-2">
//             <BarChart3 className="w-5 h-5 text-green-600" />
//           </div>
//           <div className="text-sm font-medium text-green-900">
//             Analyze All Columns
//           </div>
//           <div className="text-xs text-green-700 mt-1">
//             Complete overview
//           </div>
//         </button>

//         {/* Back to Dashboard */}
//         <button
//           onClick={onBackToDashboard}
//           disabled={isLoading}
//           className="glass-panel interactive-element p-4 rounded-lg border border-gray-200 hover:bg-gray-50 transition-all disabled:opacity-50"
//         >
//           <div className="flex items-center justify-center mb-2">
//             <ArrowLeft className="w-5 h-5 text-gray-600" />
//           </div>
//           <div className="text-sm font-medium text-gray-900">
//             Back to Dashboard
//           </div>
//           <div className="text-xs text-gray-600 mt-1">
//             Start fresh
//           </div>
//         </button>
//       </div>

//       {/* Advanced Re-analysis Options */}
//       {!hasError && (
//         <div className="border-t border-gray-200 pt-6">
//           <h4 className="text-sm font-semibold text-gray-900 mb-4">
//             Or analyze a specific column:
//           </h4>
          
//           <div className="flex flex-wrap gap-2 mb-4">
//             {columns.map((column) => (
//               <button
//                 key={column.value}
//                 onClick={() => setSelectedReAnalysisColumn(column.value)}
//                 disabled={isLoading}
//                 className={`glass-panel px-3 py-2 rounded-lg text-sm font-medium transition-all disabled:opacity-50 ${
//                   selectedReAnalysisColumn === column.value
//                     ? 'border-blue-500 bg-blue-50 text-blue-900'
//                     : 'border-gray-200 text-gray-700 hover:border-blue-300 hover:bg-blue-50'
//                 }`}
//               >
//                 {column.label}
//               </button>
//             ))}
//           </div>

//           <div className="flex items-center justify-between">
//             <button
//               onClick={handleCustomReAnalyze}
//               disabled={isLoading || selectedReAnalysisColumn === selectedColumn}
//               className="glass-panel bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center font-medium transition-colors"
//             >
//               <Zap className="w-4 h-4 mr-2" />
//               {isLoading ? 'Analyzing...' : `Analyze ${columns.find(c => c.value === selectedReAnalysisColumn)?.label || 'Selected'}`}
//             </button>

//             {selectedReAnalysisColumn === selectedColumn && (
//               <div className="text-sm text-gray-500 ml-4">
//                 ℹ️ This is the same column as before
//               </div>
//             )}
//           </div>
//         </div>
//       )}

//       {/* Loading Overlay */}
//       {isLoading && (
//         <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center rounded-lg">
//           <div className="flex items-center">
//             <RefreshCw className="w-6 h-6 animate-spin text-blue-600 mr-3" />
//             <span className="text-blue-800 font-medium">Processing analysis...</span>
//           </div>
//         </div>
//       )}


//     </div>
//   );
// };

// export default ReAnalysisPanel;