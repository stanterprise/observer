import { useEffect, useState } from 'react'
import { apiUrl } from '../lib/config'
import { Card, CardHeader, CardTitle, CardContent } from '../components/Card'
import { Badge } from '../components/Badge'
import type { TestCaseRun, WebSocketEvent } from '../types'
import { Play, Clock } from 'lucide-react'

interface TestRunsPageProps {
  onWebSocketEvent?: WebSocketEvent | null
}

export function TestRunsPage({ onWebSocketEvent }: TestRunsPageProps) {
  const [tests, setTests] = useState<TestCaseRun[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchTests()
  }, [])

  const fetchTests = async () => {
    try {
      setLoading(true)
      const response = await fetch(apiUrl('/tests?limit=50'))
      if (!response.ok) {
        throw new Error(`Failed to fetch tests: ${response.statusText}`)
      }
      const data = await response.json()
      setTests(data.tests || [])
      setError(null)
    } catch (err) {
      console.error('Error fetching tests:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch tests')
    } finally {
      setLoading(false)
    }
  }

  // Handle WebSocket events
  useEffect(() => {
    if (onWebSocketEvent) {
      // Re-fetch when we get events
      fetchTests()
    }
  }, [onWebSocketEvent])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-600">Loading tests...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-red-600">Error: {error}</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-gray-900">Test Runs</h1>
        <button
          onClick={fetchTests}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
        >
          Refresh
        </button>
      </div>

      {tests.length === 0 ? (
        <Card>
          <CardContent>
            <div className="text-center py-12">
              <Play className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No tests found</h3>
              <p className="mt-1 text-sm text-gray-500">
                Test runs will appear here once tests are executed.
              </p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {tests.map((test) => (
            <Card key={test.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{test.title || test.test_case_id}</CardTitle>
                  <Badge status={test.status} />
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-gray-600">File:</span>{' '}
                    <span className="text-gray-900 font-mono text-xs">{test.file}</span>
                  </div>
                  <div>
                    <span className="text-gray-600">Project:</span>{' '}
                    <span className="text-gray-900">{test.project}</span>
                  </div>
                  <div className="flex items-center">
                    <Clock className="h-4 w-4 mr-1 text-gray-400" />
                    <span className="text-gray-600">Started:</span>{' '}
                    <span className="text-gray-900 ml-1">
                      {new Date(test.started_at).toLocaleString()}
                    </span>
                  </div>
                  {test.finished_at && (
                    <div className="flex items-center">
                      <Clock className="h-4 w-4 mr-1 text-gray-400" />
                      <span className="text-gray-600">Finished:</span>{' '}
                      <span className="text-gray-900 ml-1">
                        {new Date(test.finished_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                </div>
                {test.error_message && (
                  <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md">
                    <p className="text-sm text-red-900 font-mono">{test.error_message}</p>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
