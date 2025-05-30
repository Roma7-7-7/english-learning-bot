import React, { useEffect, useState } from 'react';
import { Container, Card, Row, Col, Form } from 'react-bootstrap';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    BarElement,
} from 'chart.js';
import { Line, Bar } from 'react-chartjs-2';
import { format, subDays } from 'date-fns';
import client from '../api/client.tsx';
import type { DailyStats, DailyStatsRange } from '../api/client.tsx';

ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    Title,
    Tooltip,
    Legend
);

export function Stats() {
    const [dailyStats, setDailyStats] = useState<DailyStats | null>(null);
    const [rangeStats, setRangeStats] = useState<DailyStatsRange | null>(null);
    const [dateRange, setDateRange] = useState(7); // days

    useEffect(() => {
        // Fetch today's stats
        client.getDailyStats()
            .then(r => r.json())
            .then(setDailyStats)
            .catch(console.error);

        // Fetch range stats
        const to = new Date();
        const from = subDays(to, dateRange);
        client.getDailyStatsRange(from, to)
            .then(r => r.json())
            .then(setRangeStats)
            .catch(console.error);
    }, [dateRange]);

    const lineChartData = {
        labels: rangeStats?.items.map(item => format(new Date(item.date), 'MMM d')) || [],
        datasets: [
            {
                label: 'Total Words Learned',
                data: rangeStats?.items.map(item => item.total_words_learned) || [],
                borderColor: 'rgb(75, 192, 192)',
                tension: 0.1,
            },
        ],
    };

    const barChartData = {
        labels: rangeStats?.items.map(item => format(new Date(item.date), 'MMM d')) || [],
        datasets: [
            {
                label: 'Words Guessed',
                data: rangeStats?.items.map(item => item.words_guessed) || [],
                backgroundColor: 'rgba(75, 192, 192, 0.5)',
            },
            {
                label: 'Words Missed',
                data: rangeStats?.items.map(item => item.words_missed) || [],
                backgroundColor: 'rgba(255, 99, 132, 0.5)',
            },
        ],
    };

    const chartOptions = {
        responsive: true,
        plugins: {
            legend: {
                position: 'top' as const,
            },
        },
        scales: {
            y: {
                beginAtZero: true,
            },
        },
    };

    return (
        <Container>
            <h1 className="mb-4">Statistics</h1>
            
            <Row className="mb-4">
                <Col md={4}>
                    <Card>
                        <Card.Body>
                            <Card.Title>Today's Progress</Card.Title>
                            <div className="d-flex justify-content-between align-items-center">
                                <div>
                                    <div className="text-success">Guessed: {dailyStats?.words_guessed || 0}</div>
                                    <div className="text-danger">Missed: {dailyStats?.words_missed || 0}</div>
                                </div>
                                <div className="text-primary">
                                    Total Learned: {dailyStats?.total_words_learned || 0}
                                </div>
                            </div>
                        </Card.Body>
                    </Card>
                </Col>
                <Col md={8}>
                    <Form.Group>
                        <Form.Label>Time Range</Form.Label>
                        <Form.Select 
                            value={dateRange} 
                            onChange={(e) => setDateRange(Number(e.target.value))}
                        >
                            <option value={7}>Last 7 days</option>
                            <option value={14}>Last 14 days</option>
                            <option value={30}>Last 30 days</option>
                        </Form.Select>
                    </Form.Group>
                </Col>
            </Row>

            <Row className="mb-4">
                <Col>
                    <Card>
                        <Card.Body>
                            <Card.Title>Total Words Learned Over Time</Card.Title>
                            <Line data={lineChartData} options={chartOptions} />
                        </Card.Body>
                    </Card>
                </Col>
            </Row>

            <Row>
                <Col>
                    <Card>
                        <Card.Body>
                            <Card.Title>Daily Words Progress</Card.Title>
                            <Bar data={barChartData} options={chartOptions} />
                        </Card.Body>
                    </Card>
                </Col>
            </Row>
        </Container>
    );
} 