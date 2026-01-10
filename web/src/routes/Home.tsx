import { type JSX, useEffect, useState, useRef } from "react";
import client, { type Words, type WordsQueryParams } from "../api/client.tsx";
import { useAppState } from "../context.tsx";
import {
  Container,
  Row,
  Col,
  Form,
  Button,
  Table,
  Pagination,
  Alert,
  Spinner,
  Badge,
  InputGroup,
} from "react-bootstrap";
import { WordModal } from "../components/WordModal.tsx";

import { PencilSquare, Trash, X } from "react-bootstrap-icons";

interface ModalState {
  show: boolean;
  action: "add" | "edit";
  word: string;
  translation: string;
  description?: string;
}

export function Home() {
  const { refreshStats } = useAppState();
  const [words, setWords] = useState<Words | null>(null);
  const [qp, setQP] = useState({
    search: "",
    to_review: false,
    guessed: "all",
    offset: 0,
    limit: 15,
  } as WordsQueryParams);
  const [error, setError] = useState<string>("");
  const [refetchTrigger, setRefetchTrigger] = useState(0);

  const [modalState, setModalState] = useState<ModalState>({
    show: false,
    action: "add",
    word: "",
    translation: "",
    description: undefined,
  });

  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (error !== "") {
      setError("");
    }

    client
      .findWords(qp)
      .then((r) => {
        if (r.status === 200) {
          return r.json() as Promise<Words>;
        }

        throw new Error("Failed to fetch words");
      })
      .then((w) => {
        // It may happen if we applied some filtering which has words but current page overflows the total number of filtered words
        if (w.items.length == 0 && w.total > 0) {
          setQP((existing) => {
            return {
              ...existing,
              offset: 0,
            };
          });
          return;
        }

        setWords(w);
      })
      .catch((e) => {
        console.error("Error fetching words:", e);
        setError("Failed to fetch words");
        setWords(null);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [qp, refetchTrigger]);

  useEffect(() => {
    function handleKeyPress(event: KeyboardEvent) {
      // Only handle shortcuts if modal is not open
      if (!modalState.show) {
        if (
          event.key === "q" &&
          document.activeElement !== searchInputRef.current
        ) {
          event.preventDefault(); // Prevent the 'q' from being typed
          event.stopPropagation(); // Stop event from bubbling up
          setModalState({
            show: true,
            action: "add",
            word: "",
            translation: "",
            description: undefined,
          });
        } else if (event.key === "/") {
          event.preventDefault();
          if (searchInputRef.current) {
            searchInputRef.current.focus();
            searchInputRef.current.select();
          }
        } else if (event.key === "Escape") {
          if (
            document.activeElement &&
            document.activeElement !== document.body
          ) {
            (document.activeElement as HTMLElement).blur();
          }
        }
      }
    }

    document.addEventListener("keydown", handleKeyPress);
    return () => {
      document.removeEventListener("keydown", handleKeyPress);
    };
  }, [modalState.show]);

  function handleDeleteWord(word: string) {
    if (confirm(`Are you sure you want to delete the word "${word}"?`)) {
      client
        .deleteWord(word)
        .then((r) => {
          if (r.status === 200) {
            refreshStats();
            setRefetchTrigger((prev) => prev + 1);
          } else {
            setError("Failed to delete word");
          }
        })
        .catch((e) => {
          console.error("Error deleting word:", e);
          setError("Failed to delete word");
        });
    }
  }

  const onPageChange = (page: number) => {
    setQP((existing: WordsQueryParams) => {
      return {
        ...existing,
        offset: (page - 1) * existing.limit,
      };
    });
  };

  const handleCloseModal = () => {
    setModalState({
      ...modalState,
      show: false,
    });
  };

  const handleWordSuccess = () => {
    refreshStats();
    setRefetchTrigger((prev) => prev + 1);
  };

  const handleMarkToReview = (word: string, to_review: boolean) => {
    if (error !== "") {
      setError("");
    }
    if (
      !to_review &&
      !confirm("Are you sure you want to mark/unmark this word for review?")
    ) {
      return;
    }
    client
      .markToReview({ word, to_review })
      .then((r) => {
        if (r.status === 200) {
          refreshStats();
          setRefetchTrigger((prev) => prev + 1);
        } else {
          setError("Failed to mark word");
        }
      })
      .catch((e) => {
        console.error("Error marking word:", e);
        setError("Failed to mark word");
      });
  };

  const isWordLearned = (guessedStreak: number) => {
    return guessedStreak >= 15;
  };

  return (
    <>
      {!words ? (
        <Container className="text-center mt-5">
          <Spinner animation="border" role="status">
            <span className="visually-hidden">Loading...</span>
          </Spinner>
          <h1>Loading...</h1>
        </Container>
      ) : (
        <Container id="content" className="p-3">
          <Row className="mb-3 g-2">
            <Col xs={12} sm={6} lg={3}>
              <Form.Group>
                <InputGroup>
                  <Form.Control
                    ref={searchInputRef}
                    type="text"
                    placeholder="Search words..."
                    value={qp.search}
                    onChange={(present) => {
                      setQP((existing: WordsQueryParams) => {
                        return {
                          ...existing,
                          search: present.target.value,
                        };
                      });
                    }}
                  />
                  {qp.search && (
                    <Button
                      variant="outline-secondary"
                      onClick={() => {
                        setQP((existing: WordsQueryParams) => {
                          return {
                            ...existing,
                            search: "",
                          };
                        });
                      }}
                      title="Clear search"
                    >
                      <X />
                    </Button>
                  )}
                </InputGroup>
              </Form.Group>
            </Col>
            <Col xs={6} sm={3} lg={2}>
              <Form.Group>
                <Form.Select
                  value={qp.guessed}
                  onChange={(present) => {
                    setQP((existing: WordsQueryParams) => {
                      return {
                        ...existing,
                        guessed: present.target.value as
                          | "all"
                          | "learned"
                          | "batched"
                          | "to_learn",
                      };
                    });
                  }}
                >
                  <option value="all">All</option>
                  <option value="learned">Learned</option>
                  <option value="batched">Batched</option>
                  <option value="to_learn">To Learn</option>
                </Form.Select>
              </Form.Group>
            </Col>
            <Col xs={6} sm={3} lg={2} className="d-flex align-items-center">
              <Form.Check
                type="checkbox"
                id="to-review-checkbox"
                label="To Review"
                checked={qp.to_review}
                onChange={(present) => {
                  setQP((existing: WordsQueryParams) => {
                    return {
                      ...existing,
                      to_review: present.target.checked,
                    };
                  });
                }}
              />
            </Col>
            <Col xs={12} sm={12} lg={3}></Col>
            <Col xs={6} sm={3} lg={1}>
              <Button
                variant="outline-secondary"
                className="w-100"
                title="Clear filters"
                onClick={() => {
                  setQP({
                    search: "",
                    to_review: false,
                    guessed: "all",
                    offset: 0,
                    limit: 15,
                  });
                }}
              >
                <span className="d-none d-sm-inline">Clear</span>
                <span className="d-sm-none" aria-hidden="true">
                  &times;
                </span>
              </Button>
            </Col>
            <Col xs={6} sm={3} lg={1}>
              <Button
                variant="primary"
                className="w-100"
                onClick={() => {
                  setModalState({
                    show: true,
                    action: "add",
                    word: "",
                    translation: "",
                    description: undefined,
                  });
                }}
              >
                <span className="d-none d-sm-inline">Add</span>
                <span className="d-sm-none">+</span>
              </Button>
            </Col>
          </Row>

          <div id="words">
            <Row>
              <Col xs={12}>
                <div className="table-responsive">
                  <Table hover>
                    <thead>
                      <tr>
                        <th>
                          <span className="d-none d-sm-inline">Word</span>
                          <span className="d-sm-none">Word (tap to edit)</span>
                        </th>
                        <th className="d-none d-md-table-cell">Translation</th>
                        <th className="text-center">
                          <span className="d-none d-sm-inline">Learned</span>
                          <span className="d-sm-none">✓</span>
                        </th>
                        <th className="text-center">
                          <span className="d-none d-sm-inline">To Review</span>
                          <span className="d-sm-none">R</span>
                        </th>
                        <th className="text-center d-none d-sm-table-cell">
                          Edit
                        </th>
                        <th className="text-center">
                          <span className="d-none d-sm-inline">Delete</span>
                          <span className="d-sm-none">D</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {words.items.map((item) => (
                        <tr key={item.word}>
                          <td className="text-break d-none d-sm-table-cell">
                            {item.word}
                          </td>
                          <td
                            className="text-break d-sm-none"
                            role="button"
                            style={{ cursor: "pointer" }}
                            onClick={() => {
                              setModalState({
                                show: true,
                                action: "edit",
                                word: item.word,
                                translation: item.translation,
                                description: item.description,
                              });
                            }}
                            title="Tap to edit"
                          >
                            {item.word}
                          </td>
                          <td className="d-none d-md-table-cell text-break">
                            {item.translation}
                          </td>
                          <td className="text-center">
                            {isWordLearned(item.guessed_streak || 0) ? (
                              <Badge
                                bg="success"
                                title={`Streak: ${item.guessed_streak}`}
                              >
                                <span className="d-none d-sm-inline">
                                  ✓ Learned
                                </span>
                                <span className="d-sm-none">✓</span>
                              </Badge>
                            ) : (
                              <Badge
                                bg="secondary"
                                title={`Streak: ${item.guessed_streak || 0}/15`}
                              >
                                <span className="d-none d-sm-inline">
                                  {item.guessed_streak || 0}/15
                                </span>
                                <span className="d-sm-none">
                                  {item.guessed_streak || 0}
                                </span>
                              </Badge>
                            )}
                          </td>
                          <td className="text-center">
                            <Form.Check
                              type="checkbox"
                              id={`to-review-${item.word}`}
                              checked={item.to_review}
                              onChange={(present) => {
                                handleMarkToReview(
                                  item.word,
                                  present.target.checked,
                                );
                              }}
                            />
                          </td>
                          <td className="text-center d-none d-sm-table-cell">
                            <Button
                              variant="link"
                              size="sm"
                              className="p-1"
                              onClick={() => {
                                setModalState({
                                  show: true,
                                  action: "edit",
                                  word: item.word,
                                  translation: item.translation,
                                  description: item.description,
                                });
                              }}
                            >
                              <PencilSquare />
                            </Button>
                          </td>
                          <td className="text-center">
                            <Button
                              variant="link"
                              size="sm"
                              className="p-1"
                              onClick={() => handleDeleteWord(item.word)}
                            >
                              <Trash />
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </div>
              </Col>
            </Row>
            <Row>
              <Col xs={12}>
                <div className="d-flex justify-content-center">
                  {paginationFooter(qp, words.total, onPageChange)}
                </div>
              </Col>
            </Row>
          </div>

          {error && (
            <Row>
              <Col>
                <Alert variant="danger">{error}</Alert>
              </Col>
            </Row>
          )}
        </Container>
      )}
      <WordModal
        show={modalState.show}
        action={modalState.action}
        word={modalState.word}
        translation={modalState.translation}
        description={modalState.description}
        onHide={handleCloseModal}
        onSuccess={handleWordSuccess}
      />
    </>
  );
}

function paginationFooter(
  qp: WordsQueryParams,
  totalItems: number,
  onPageChange: (page: number) => void,
): JSX.Element {
  const totalPages = Math.ceil(totalItems / qp.limit);
  const page = getPage(qp);
  if (totalPages <= 1) {
    return <></>;
  }

  interface PaginationItem {
    active: boolean;
    disabled: boolean;
    page: number;
    isArrow?: boolean;
  }

  const items: PaginationItem[] = [];

  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) {
      items.push({
        active: i === page,
        disabled: false,
        page: i,
      });
    }
  } else {
    items.push({
      active: false,
      disabled: page === 1,
      page: 1,
      isArrow: true,
    });

    if (page > 2) {
      items.push({
        active: false,
        disabled: false,
        page: page - 2,
      });
    }

    if (page > 1) {
      items.push({
        active: false,
        disabled: false,
        page: page - 1,
      });
    }

    items.push({
      active: true,
      disabled: true,
      page: page,
    });

    if (page < totalPages) {
      items.push({
        active: false,
        disabled: false,
        page: page + 1,
      });
    }

    if (page < totalPages - 1) {
      items.push({
        active: false,
        disabled: false,
        page: page + 2,
      });
    }

    items.push({
      active: false,
      disabled: page === totalPages,
      page: totalPages,
      isArrow: true,
    });
  }

  return (
    <Pagination>
      {items.map((item, idx) => (
        <Pagination.Item
          key={"page-" + idx}
          active={item.active}
          disabled={item.disabled}
          onClick={() => {
            if (!item.disabled) {
              onPageChange(item.page);
            }
          }}
        >
          {item.isArrow && idx === 0 && <span aria-hidden="true">&laquo;</span>}
          {!item.isArrow && item.page}
          {item.isArrow && idx === items.length - 1 && (
            <span aria-hidden="true">&raquo;</span>
          )}
        </Pagination.Item>
      ))}
    </Pagination>
  );
}

function getPage(qp: WordsQueryParams): number {
  return Math.floor(qp.offset / qp.limit) + 1;
}
